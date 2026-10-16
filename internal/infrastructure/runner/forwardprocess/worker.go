package forwardprocess

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"time"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
)

// Run executes the internal child entry. The composition root supplies the
// application usecase; this adapter owns only local process/control resources.
func (m *Manager) Run(ctx context.Context, port int, runId string, run func(context.Context, environmentdto.ForwardInput, func(environmentdto.ForwardSpec) error) error) error {
	state, err := m.startupRecord(ctx, port, runId)
	if err != nil {
		return err
	}
	state.Pid, state.StartedAt = os.Getpid(), time.Now().UTC()
	state.Created, err = processCreated(state.Pid)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer func() { _ = listener.Close() }()
	state.Control = listener.Addr().String()
	if err := m.write(state); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var mu sync.Mutex
	controlDone := make(chan struct{})
	go func() {
		defer close(controlDone)
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = connection.Close() }()
				_ = connection.SetDeadline(time.Now().Add(2 * time.Second))
				var request controlRequest
				if json.NewDecoder(io.LimitReader(connection, 4096)).Decode(&request) != nil || request.RunId != runId ||
					subtle.ConstantTimeCompare([]byte(request.Key), []byte(state.ControlKey)) != 1 {
					return
				}
				if request.Action != "status" && request.Action != "stop" {
					return
				}
				mu.Lock()
				reply := state
				if ctx.Err() != nil {
					reply.State = "stopping"
				}
				mu.Unlock()
				if json.NewEncoder(connection).Encode(reply) == nil && request.Action == "stop" {
					cancel()
				}
			}()
		}
	}()
	err = run(ctx, state.ForwardInput, func(actual environmentdto.ForwardSpec) error {
		mu.Lock()
		defer mu.Unlock()
		if err := ctx.Err(); err != nil {
			return err
		}
		if actual != state.ForwardSpec {
			return errors.New("environment or access identity changed during forward startup")
		}
		state.State = "running"
		return m.write(state)
	})
	_ = listener.Close()
	<-controlDone
	mu.Lock()
	defer mu.Unlock()
	state.State = "stopped"
	if err != nil && !errors.Is(err, context.Canceled) {
		state.State, state.Reason = "failed", err.Error()
	}
	if writeErr := m.write(state); err == nil {
		err = writeErr
	}
	return err
}

func (m *Manager) startupRecord(ctx context.Context, port int, runId string) (record, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		state, err := m.read(port)
		if err != nil {
			return record{}, err
		}
		if state.RunId != runId || state.State != "starting" || state.Executable != m.executable {
			return record{}, errors.New("forward worker has no matching startup record")
		}
		if state.Pid == os.Getpid() {
			matches, err := processMatches(state)
			if err != nil || !matches {
				return record{}, errors.New("forward worker OS identity does not match")
			}
			return state, nil
		}
		select {
		case <-ctx.Done():
			return record{}, errors.New("forward parent did not publish the child OS identity")
		case <-ticker.C:
		}
	}
}
