package forwardprocess

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

var _ environmentport.ForwardProcesses = (*Manager)(nil)

var errProcessExited = errors.New("process exited")

type Manager struct {
	directory, executable, cwd string
}

type record struct {
	environmentdto.ForwardStatus
	RunId      string `json:"run_id"`
	ControlKey string `json:"control_key"`
	Executable string `json:"executable"`
	Created    string `json:"process_created"`
	Control    string `json:"control"`
}

type controlRequest struct {
	RunId  string `json:"run_id"`
	Key    string `json:"key"`
	Action string `json:"action"`
}

func New(directory string) (*Manager, error) {
	current, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("identify local user: %w", err)
	}
	owner := sha256.Sum256([]byte(current.Uid))
	directory, err = filepath.Abs(filepath.Join(directory, hex.EncodeToString(owner[:8])))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, err
	}
	if err := protectDirectory(directory, current.Uid); err != nil {
		return nil, err
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return &Manager{directory: directory, executable: executable, cwd: cwd}, nil
}

func (m *Manager) path(port int) string {
	return filepath.Join(m.directory, strconv.Itoa(port)+".json")
}

func (m *Manager) read(port int) (record, error) {
	file, err := os.Open(m.path(port))
	if err != nil {
		return record{}, err
	}
	defer func() { _ = file.Close() }()
	var state record
	err = json.NewDecoder(io.LimitReader(file, 64*1024)).Decode(&state)
	if err == nil && (state.LocalPort != port || port < 1 || port > 65535 || len(state.RunId) != 64 || len(state.ControlKey) != 64) {
		err = errors.New("invalid forward runtime record")
	}
	return state, err
}

func (m *Manager) write(state record) error {
	file, err := os.CreateTemp(m.directory, ".forward-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(file.Name()) }()
	if err := json.NewEncoder(file).Encode(state); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), m.path(state.LocalPort))
}

func validPort(port int) error {
	if port < 1 || port > 65535 {
		return errors.New("local port must be 1–65535")
	}
	return nil
}

func (m *Manager) lock(ctx context.Context, port int) (func(), error) {
	if err := validPort(port); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(m.directory, strconv.Itoa(port)+".lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	for {
		locked, err := tryLock(file)
		if err != nil {
			_ = file.Close()
			return nil, err
		}
		if locked {
			return func() { _ = file.Close() }, nil
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			_ = file.Close()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (m *Manager) control(ctx context.Context, state record, action string) (record, error) {
	host, port, err := net.SplitHostPort(state.Control)
	if err != nil || host != "127.0.0.1" || port == "0" {
		return record{}, errors.New("forward control is unavailable")
	}
	created, err := processCreated(state.Pid)
	if err != nil || created != state.Created || state.Executable != m.executable {
		return record{}, errors.New("forward process identity no longer matches")
	}
	connection, err := (&net.Dialer{Timeout: time.Second}).DialContext(ctx, "tcp4", state.Control)
	if err != nil {
		return record{}, err
	}
	defer func() { _ = connection.Close() }()
	deadline := time.Now().Add(2 * time.Second)
	if end, ok := ctx.Deadline(); ok && end.Before(deadline) {
		deadline = end
	}
	if err := connection.SetDeadline(deadline); err != nil {
		return record{}, err
	}
	stop := context.AfterFunc(ctx, func() { _ = connection.Close() })
	defer stop()
	if err := json.NewEncoder(connection).Encode(controlRequest{RunId: state.RunId, Key: state.ControlKey, Action: action}); err != nil {
		return record{}, err
	}
	var reply record
	if err := json.NewDecoder(io.LimitReader(connection, 64*1024)).Decode(&reply); err != nil {
		return record{}, err
	}
	if reply.RunId != state.RunId || reply.Pid != state.Pid || reply.Created != state.Created || reply.Executable != state.Executable || reply.ForwardSpec != state.ForwardSpec {
		return record{}, errors.New("forward control identity no longer matches")
	}
	return reply, nil
}

func (m *Manager) inspect(ctx context.Context, state record) environmentdto.ForwardStatus {
	if live, err := m.control(ctx, state, "status"); err == nil {
		return live.ForwardStatus
	}
	matches, err := processMatches(state)
	if err == nil && !matches {
		if state.State != "failed" && state.State != "stopped" {
			state.State, state.Reason = "stale", "background process has exited"
		}
	} else {
		state.State, state.Reason = "unresponsive", "process identity or local control could not be verified"
	}
	return state.ForwardStatus
}

func processMatches(state record) (bool, error) {
	created, err := processCreated(state.Pid)
	if errors.Is(err, errProcessExited) {
		return false, nil
	}
	return err == nil && created == state.Created, err
}

func (m *Manager) Status(ctx context.Context, port int) ([]environmentdto.ForwardStatus, error) {
	ports := []int{port}
	if port == 0 {
		paths, err := filepath.Glob(filepath.Join(m.directory, "*.json"))
		if err != nil {
			return nil, err
		}
		ports = nil
		for _, path := range paths {
			value, err := strconv.Atoi(strings.TrimSuffix(filepath.Base(path), ".json"))
			if err != nil || validPort(value) != nil {
				continue
			}
			ports = append(ports, value)
		}
		sort.Ints(ports)
	} else if err := validPort(port); err != nil {
		return nil, err
	}
	result := make([]environmentdto.ForwardStatus, 0, len(ports))
	for _, current := range ports {
		state, err := m.read(current)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read forward %d: %w", current, err)
		}
		result = append(result, m.inspect(ctx, state))
	}
	return result, nil
}

func (m *Manager) Start(ctx context.Context, spec environmentdto.ForwardSpec) (environmentdto.ForwardStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	unlock, err := m.lock(ctx, spec.LocalPort)
	if err != nil {
		return environmentdto.ForwardStatus{}, err
	}
	defer unlock()
	previous, err := m.read(spec.LocalPort)
	if err == nil {
		status := m.inspect(ctx, previous)
		if status.State == "running" && status.ForwardSpec == spec {
			return status, nil
		}
		if status.State == "running" || status.State == "starting" || status.State == "stopping" || status.State == "unresponsive" {
			return environmentdto.ForwardStatus{}, fmt.Errorf("local port %d already has a managed forward (%s)", spec.LocalPort, status.State)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return environmentdto.ForwardStatus{}, err
	}
	random := make([]byte, 64)
	if _, err := rand.Read(random); err != nil {
		return environmentdto.ForwardStatus{}, err
	}
	// The run ID identifies the child on argv; control authentication stays in
	// the private runtime record and must never be exposed in process arguments.
	state := record{ForwardStatus: environmentdto.ForwardStatus{ForwardSpec: spec, State: "starting"},
		RunId: hex.EncodeToString(random[:32]), ControlKey: hex.EncodeToString(random[32:]), Executable: m.executable}
	if err := m.write(state); err != nil {
		return environmentdto.ForwardStatus{}, err
	}
	logPath := filepath.Join(m.directory, strconv.Itoa(spec.LocalPort)+".log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		_ = os.Remove(m.path(spec.LocalPort))
		return environmentdto.ForwardStatus{}, err
	}
	command := exec.Command(m.executable, "cli", "env", "forward", "__run", "--local-port", strconv.Itoa(spec.LocalPort), "--run-id", state.RunId)
	command.Dir, command.Stdout, command.Stderr = m.cwd, logFile, logFile
	detach(command)
	err = command.Start()
	_ = logFile.Close()
	if err != nil {
		_ = os.Remove(m.path(spec.LocalPort))
		return environmentdto.ForwardStatus{}, err
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	started := false
	defer func() {
		if !started {
			select {
			case <-done:
			default:
				_ = command.Process.Kill()
				<-done
			}
			_ = os.Remove(m.path(spec.LocalPort))
		}
	}()
	// Publish the OS identity before the child can acquire its runtime record.
	// A crashed parent cannot leave an unidentifiable running child behind.
	state.Pid = command.Process.Pid
	state.Created, err = processCreated(state.Pid)
	if err != nil {
		return environmentdto.ForwardStatus{}, fmt.Errorf("identify forward worker: %w", err)
	}
	if err := m.write(state); err != nil {
		return environmentdto.ForwardStatus{}, err
	}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return environmentdto.ForwardStatus{}, fmt.Errorf("forward startup cancelled: %w", ctx.Err())
		case err := <-done:
			// Keep the completion available to the failed-start cleanup owner.
			done <- err
			current, readErr := m.read(spec.LocalPort)
			if readErr == nil && current.RunId == state.RunId && current.Reason != "" {
				return environmentdto.ForwardStatus{}, errors.New(current.Reason)
			}
			return environmentdto.ForwardStatus{}, fmt.Errorf("forward worker exited (%v); see %s", err, logPath)
		case <-ticker.C:
			current, err := m.read(spec.LocalPort)
			if err != nil || current.RunId != state.RunId {
				continue
			}
			if current.State == "running" {
				live, err := m.control(ctx, current, "status")
				if err == nil && live.State == "running" && live.Pid == command.Process.Pid {
					started = true
					return live.ForwardStatus, nil
				}
			}
		}
	}
}

func (m *Manager) Stop(ctx context.Context, port int) error {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	unlock, err := m.lock(ctx, port)
	if err != nil {
		return err
	}
	defer unlock()
	state, err := m.read(port)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	matches, err := processMatches(state)
	if err != nil {
		return fmt.Errorf("cannot inspect forward process: %w", err)
	}
	if !matches {
		return os.Remove(m.path(port))
	}
	if state.State != "stopped" && state.State != "failed" {
		if _, err := m.control(ctx, state, "stop"); err != nil {
			return fmt.Errorf("cannot verify/stop managed forward: %w", err)
		}
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		matches, err := processMatches(state)
		if err != nil {
			return fmt.Errorf("cannot inspect forward process: %w", err)
		}
		if !matches {
			return os.Remove(m.path(port))
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("forward has not stopped: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}
