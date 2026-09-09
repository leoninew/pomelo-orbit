package sshrunner

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const maxRemoteErrorOutputBytes = 4 * 1024

type Runtime struct {
	dialContext func(context.Context, string, string) (net.Conn, error)
	locks       sync.Map
}

func NewRuntime() *Runtime {
	dialer := net.Dialer{}
	return &Runtime{dialContext: dialer.DialContext}
}

func (r *Runtime) ServiceDir(target environmentport.Target, serviceCode string) (string, error) {
	if !target.Environment.IsSSH() {
		return "", errors.New("SSH deployment runtime received a non-SSH environment")
	}
	if !safePathSegment(serviceCode) {
		return "", errors.New("invalid service code")
	}
	root := normalizeRemotePath(target.Environment.SSH.WorkspaceRoot)
	if root == "" {
		return "", errors.New("environment workspace root is required")
	}
	return path.Join(root, serviceCode), nil
}

func (r *Runtime) ComposeMountSourceDir(_ context.Context, target environmentport.Target, serviceCode string) (string, error) {
	if _, err := r.ServiceDir(target, serviceCode); err != nil {
		return "", err
	}
	return "", nil
}

func (r *Runtime) ServiceDirExists(ctx context.Context, target environmentport.Target, serviceCode string) (bool, error) {
	serviceDir, err := r.ServiceDir(target, serviceCode)
	if err != nil {
		return false, err
	}
	client, cleanup, err := r.openSFTP(ctx, target)
	if err != nil {
		return false, err
	}
	defer cleanup()
	serviceDir, err = newSFTPPathResolver(client).resolve(serviceDir)
	if err != nil {
		return false, err
	}
	info, err := client.Stat(serviceDir)
	if err == nil {
		return info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("inspect remote service workspace: %w", err)
}

func (r *Runtime) StageWorkspace(ctx context.Context, target environmentport.Target, workspace deploymentport.Workspace) error {
	logicalServiceDir, err := r.ServiceDir(target, workspace.ServiceCode)
	if err != nil {
		return err
	}
	unlock := r.lock(target.Environment.Id + "\x00" + workspace.ServiceCode)
	defer unlock()

	client, cleanup, err := r.openSFTP(ctx, target)
	if err != nil {
		return err
	}
	defer cleanup()
	pathResolver := newSFTPPathResolver(client)
	serviceDir, err := pathResolver.resolve(logicalServiceDir)
	if err != nil {
		return err
	}
	if err := client.MkdirAll(serviceDir); err != nil {
		return fmt.Errorf("create remote service workspace: %w", err)
	}
	for _, directory := range workspace.Directories {
		directory, err = pathResolver.resolve(directory)
		if err != nil {
			return err
		}
		if err := client.MkdirAll(directory); err != nil {
			return fmt.Errorf("create remote mount directory: %w", err)
		}
	}
	for _, file := range workspace.Files {
		file.Path, err = pathResolver.resolve(file.Path)
		if err != nil {
			return err
		}
		if err := writeWorkspaceFile(client, target.Environment.SSH.Platform, file.Path, file.Content, file.Mode, file.IgnoreIfExists, workspace.DeploymentID); err != nil {
			return err
		}
	}
	composePath := path.Join(serviceDir, "docker-compose.yml")
	compose := strings.ReplaceAll(workspace.Compose, logicalServiceDir, serviceDir)
	if err := writeWorkspaceFile(client, target.Environment.SSH.Platform, composePath, []byte(compose), 0o644, false, workspace.DeploymentID); err != nil {
		return err
	}
	return nil
}

func (r *Runtime) Run(ctx context.Context, target environmentport.Target, serviceCode string, log io.Writer, name string, args ...string) error {
	serviceDir, err := r.ServiceDir(target, serviceCode)
	if err != nil {
		return err
	}
	command, display, err := remoteCommand(target.Environment.SSH.Platform, serviceDir, name, args...)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(log, "Running: %s\n", display); err != nil {
		return err
	}
	client, cleanup, err := r.openSSH(ctx, target)
	if err != nil {
		return err
	}
	defer cleanup()
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create SSH session: %w", err)
	}
	defer func() { _ = session.Close() }()
	var output bytes.Buffer
	writer := io.MultiWriter(log, &output)
	session.Stdout = writer
	session.Stderr = writer
	if err := session.Run(command); err != nil {
		if detail := boundedRemoteOutput(output.String()); detail != "" {
			return fmt.Errorf("run remote %s: %w\n%s", display, err, detail)
		}
		return fmt.Errorf("run remote %s: %w", display, err)
	}
	return nil
}

func (r *Runtime) Query(ctx context.Context, target environmentport.Target, serviceCode string, name string, args ...string) (string, error) {
	serviceDir, err := r.ServiceDir(target, serviceCode)
	if err != nil {
		return "", err
	}
	return r.queryAt(ctx, target, serviceDir, name, args...)
}

func (r *Runtime) QueryAtEnvironmentRoot(ctx context.Context, target environmentport.Target, name string, args ...string) (string, error) {
	if !target.Environment.IsSSH() {
		return "", errors.New("SSH deployment runtime received a non-SSH environment")
	}
	root := normalizeRemotePath(target.Environment.SSH.WorkspaceRoot)
	if root == "" {
		return "", errors.New("environment workspace root is required")
	}
	return r.queryAt(ctx, target, root, name, args...)
}

func (r *Runtime) queryAt(ctx context.Context, target environmentport.Target, workingDirectory string, name string, args ...string) (string, error) {
	if !target.Environment.IsSSH() {
		return "", errors.New("SSH deployment runtime received a non-SSH environment")
	}
	command, _, err := remoteCommand(target.Environment.SSH.Platform, workingDirectory, name, args...)
	if err != nil {
		return "", err
	}
	client, cleanup, err := r.openSSH(ctx, target)
	if err != nil {
		return "", err
	}
	defer cleanup()
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("create SSH session: %w", err)
	}
	defer func() { _ = session.Close() }()
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	if err := session.Run(command); err != nil {
		return remoteQueryDiagnostic(stdout.String(), stderr.String()), err
	}
	return stdout.String(), nil
}

func (r *Runtime) openSFTP(ctx context.Context, target environmentport.Target) (*sftp.Client, func(), error) {
	sshClient, closeSSH, err := r.openSSH(ctx, target)
	if err != nil {
		return nil, nil, err
	}
	client, err := sftp.NewClient(sshClient)
	if err != nil {
		closeSSH()
		return nil, nil, fmt.Errorf("start SFTP client: %w", err)
	}
	return client, func() {
		_ = client.Close()
		closeSSH()
	}, nil
}

func (r *Runtime) openSSH(ctx context.Context, target environmentport.Target) (*ssh.Client, func(), error) {
	if r == nil || r.dialContext == nil {
		return nil, nil, errors.New("SSH runtime dialer is not configured")
	}
	if !target.Environment.IsSSH() || target.PrivateKey == nil {
		return nil, nil, errors.New("SSH deployment runtime received an invalid SSH target")
	}
	signer, err := parseSigner(*target.PrivateKey)
	if err != nil {
		return nil, nil, err
	}
	environment := target.Environment.SSH
	address := net.JoinHostPort(strings.TrimSpace(environment.Host), strconv.Itoa(environment.Port))
	connection, err := r.dialContext(ctx, "tcp", address)
	if err != nil {
		return nil, nil, fmt.Errorf("dial SSH target: %w", err)
	}
	stopCancelClose := context.AfterFunc(ctx, func() { _ = connection.Close() })
	clientConfig := &ssh.ClientConfig{
		User:            environment.Username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: strictHostKeyCallback(environment.HostKeyFingerprint),
	}
	clientConnection, channels, requests, err := ssh.NewClientConn(connection, address, clientConfig)
	if err != nil {
		stopCancelClose()
		_ = connection.Close()
		return nil, nil, fmt.Errorf("establish SSH connection: %w", err)
	}
	client := ssh.NewClient(clientConnection, channels, requests)
	return client, func() {
		stopCancelClose()
		_ = client.Close()
	}, nil
}

func (r *Runtime) lock(key string) func() {
	value, _ := r.locks.LoadOrStore(key, &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	return mutex.Unlock
}

func writeWorkspaceFile(client *sftp.Client, platform string, filePath string, content []byte, mode uint32, ignoreIfExists bool, revision string) error {
	filePath = normalizeRemotePath(filePath)
	if filePath == "" {
		return errors.New("remote file path is required")
	}
	if info, err := client.Stat(filePath); err == nil {
		if info.IsDir() {
			return fmt.Errorf("remote path %s is a directory", filePath)
		}
		if ignoreIfExists {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect remote file: %w", err)
	}
	if err := client.MkdirAll(path.Dir(filePath)); err != nil {
		return fmt.Errorf("create remote file parent: %w", err)
	}
	tempPath := filePath + ".orbit-" + safeTempSuffix(revision)
	file, err := client.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return fmt.Errorf("open remote staged file: %w", err)
	}
	_, writeErr := file.Write(content)
	closeErr := file.Close()
	if writeErr != nil {
		_ = client.Remove(tempPath)
		return fmt.Errorf("write remote staged file: %w", writeErr)
	}
	if closeErr != nil {
		_ = client.Remove(tempPath)
		return fmt.Errorf("close remote staged file: %w", closeErr)
	}
	if platform == model.EnvironmentPlatformLinux {
		if mode == 0 {
			mode = 0o644
		}
		if err := client.Chmod(tempPath, os.FileMode(mode)); err != nil {
			_ = client.Remove(tempPath)
			return fmt.Errorf("set remote file mode: %w", err)
		}
	}
	if err := client.PosixRename(tempPath, filePath); err != nil {
		_ = client.Remove(tempPath)
		return fmt.Errorf("commit remote staged file: %w", err)
	}
	return nil
}

func remoteCommand(platform string, workingDirectory string, name string, args ...string) (string, string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, "\r\n") {
		return "", "", errors.New("remote command is required")
	}
	displayParts := append([]string{name}, args...)
	display := strings.Join(displayParts, " ")
	switch platform {
	case model.EnvironmentPlatformLinux:
		parts := make([]string, 0, len(args)+1)
		parts = append(parts, posixQuote(name))
		for _, arg := range args {
			parts = append(parts, posixRemoteArgument(arg))
		}
		script := "cd " + posixRemotePath(workingDirectory) + " && exec " + strings.Join(parts, " ")
		return "sh -lc " + posixQuote(script), display, nil
	case model.EnvironmentPlatformWindows:
		psArgs := make([]string, 0, len(args))
		for _, arg := range args {
			psArgs = append(psArgs, powerShellRemoteArgument(arg))
		}
		commandName := name
		switch {
		case strings.EqualFold(name, "docker"):
			commandName = "docker.exe"
		case strings.EqualFold(name, "curl"):
			commandName = "curl.exe"
		}
		script := "$ErrorActionPreference = 'Stop'; Set-Location -LiteralPath " + powerShellRemotePath(workingDirectory) + "; & " + powerShellQuote(commandName)
		if len(psArgs) > 0 {
			script += " @(" + strings.Join(psArgs, ",") + ")"
		}
		script += "; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }"
		return powerShellEncodedCommand(script), display, nil
	default:
		return "", "", errors.New("unsupported environment platform")
	}
}

func normalizeRemotePath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return ""
	}
	return path.Clean(value)
}

type sftpPathResolver struct {
	client *sftp.Client
	home   string
}

func newSFTPPathResolver(client *sftp.Client) *sftpPathResolver {
	return &sftpPathResolver{client: client}
}

func (r *sftpPathResolver) resolve(value string) (string, error) {
	value = normalizeRemotePath(value)
	if value == "" {
		return "", errors.New("remote path is required")
	}
	if !isRemoteHomePath(value) {
		return value, nil
	}
	if r.home == "" {
		home, err := r.client.RealPath(".")
		if err != nil {
			return "", fmt.Errorf("resolve remote SSH home directory: %w", err)
		}
		r.home = normalizeRemotePath(home)
		if r.home == "" {
			return "", errors.New("remote SSH home directory is empty")
		}
	}
	if value == "~" {
		return r.home, nil
	}
	return path.Join(r.home, strings.TrimPrefix(value, "~/")), nil
}

func isRemoteHomePath(value string) bool {
	return value == "~" || strings.HasPrefix(value, "~/")
}

func posixRemotePath(value string) string {
	value = normalizeRemotePath(value)
	if !isRemoteHomePath(value) {
		return posixQuote(value)
	}
	if value == "~" {
		return `"$HOME"`
	}
	return `"$HOME"` + posixQuote(strings.TrimPrefix(value, "~"))
}

func posixRemoteArgument(value string) string {
	if strings.HasPrefix(value, "@") && isRemoteHomePath(strings.TrimPrefix(value, "@")) {
		return posixQuote("@") + posixRemotePath(strings.TrimPrefix(value, "@"))
	}
	if isRemoteHomePath(value) {
		return posixRemotePath(value)
	}
	return posixQuote(value)
}

func powerShellRemotePath(value string) string {
	value = normalizeRemotePath(value)
	if !isRemoteHomePath(value) {
		return powerShellQuote(value)
	}
	if value == "~" {
		return "$HOME"
	}
	return "(Join-Path -Path $HOME -ChildPath " + powerShellQuote(strings.TrimPrefix(value, "~/")) + ")"
}

func powerShellRemoteArgument(value string) string {
	if strings.HasPrefix(value, "@") && isRemoteHomePath(strings.TrimPrefix(value, "@")) {
		return "('@' + " + powerShellRemotePath(strings.TrimPrefix(value, "@")) + ")"
	}
	if isRemoteHomePath(value) {
		return powerShellRemotePath(value)
	}
	return powerShellQuote(value)
}

func safePathSegment(value string) bool {
	return value != "" && value != "." && value != ".." && !strings.ContainsAny(value, "/\\\r\n")
}

func safeTempSuffix(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return "staged"
	}
	return builder.String()
}

func posixQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func powerShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

const powerShellEncodedCommandPrefix = "powershell.exe -NoProfile -NonInteractive -OutputFormat Text -EncodedCommand "

func powerShellEncodedCommand(script string) string {
	script = "$ProgressPreference = 'SilentlyContinue'; " + script
	codeUnits := utf16.Encode([]rune(script))
	bytes := make([]byte, len(codeUnits)*2)
	for index, codeUnit := range codeUnits {
		bytes[index*2] = byte(codeUnit)
		bytes[index*2+1] = byte(codeUnit >> 8)
	}
	return powerShellEncodedCommandPrefix + base64.StdEncoding.EncodeToString(bytes)
}

func remoteQueryDiagnostic(stdout string, stderr string) string {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return stdout
	}
	if strings.TrimSpace(stdout) == "" {
		return stderr
	}
	return stdout + "\n" + stderr
}

func boundedRemoteOutput(output string) string {
	output = strings.TrimSpace(output)
	if len(output) <= maxRemoteErrorOutputBytes {
		return output
	}
	return "...\n" + output[len(output)-maxRemoteErrorOutputBytes:]
}

func (r *Runtime) SyncFiles(ctx context.Context, target environmentport.Target, directory string, files []deploymentport.WorkspaceFile, pruneSuffix string) error {
	directory = normalizeRemotePath(directory)
	if directory == "" {
		return errors.New("remote sync directory is required")
	}
	unlock := r.lock(target.Environment.Id + "\x00" + directory)
	defer unlock()
	client, cleanup, err := r.openSFTP(ctx, target)
	if err != nil {
		return err
	}
	defer cleanup()
	pathResolver := newSFTPPathResolver(client)
	directory, err = pathResolver.resolve(directory)
	if err != nil {
		return err
	}
	if err := client.MkdirAll(directory); err != nil {
		return fmt.Errorf("create remote sync directory: %w", err)
	}
	keep := make(map[string]struct{}, len(files))
	for _, file := range files {
		file.Path, err = pathResolver.resolve(file.Path)
		if err != nil {
			return err
		}
		if path.Dir(file.Path) != directory || !safePathSegment(path.Base(file.Path)) {
			return errors.New("remote sync file must be a direct child of the sync directory")
		}
		keep[path.Base(file.Path)] = struct{}{}
		if err := writeWorkspaceFile(client, target.Environment.SSH.Platform, file.Path, file.Content, file.Mode, file.IgnoreIfExists, "sync"); err != nil {
			return err
		}
	}
	if pruneSuffix == "" {
		return nil
	}
	entries, err := client.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("inspect remote sync directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), pruneSuffix) {
			continue
		}
		if _, found := keep[entry.Name()]; found {
			continue
		}
		if err := client.Remove(path.Join(directory, entry.Name())); err != nil {
			return fmt.Errorf("prune remote sync file: %w", err)
		}
	}
	return nil
}
