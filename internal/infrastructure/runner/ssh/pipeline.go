package sshrunner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	pipelinerunport "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/port"
	"github.com/leoninew/pomelo-orbit/internal/common/workspacepath"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var _ pipelinerunport.RemoteRuntimeFactory = (*PipelineRuntime)(nil)

type PipelineRuntime struct{ ssh *Runtime }

func NewPipelineRuntime() *PipelineRuntime { return &PipelineRuntime{ssh: NewRuntime()} }

func (runtime *PipelineRuntime) RuntimeForTarget(ctx context.Context, target environmentport.Target) (pipelinerunport.Workspace, pipelinerunport.ContainerRunner, pipelinerunport.ExecutionLogStore, error) {
	if !target.Environment.IsSSH() || target.PrivateKey == nil ||
		(target.Environment.SSH.Platform != model.EnvironmentPlatformWindows && target.Environment.SSH.Platform != model.EnvironmentPlatformLinux) {
		return nil, nil, nil, errors.New("SSH pipeline target requires a supported platform and managed credential")
	}
	client, closeClient, err := runtime.ssh.openSFTP(ctx, target)
	if err != nil {
		return nil, nil, nil, err
	}
	root, err := newSFTPPathResolver(client, target.Environment.SSH.Platform).resolve(target.Environment.WorkspaceRoot)
	closeClient()
	if err != nil {
		return nil, nil, nil, err
	}
	if target.Environment.SSH.Platform == model.EnvironmentPlatformWindows && !windowsDrivePath.MatchString(root) {
		return nil, nil, nil, errors.New("windows SSH pipeline workspace must resolve to a drive-absolute path")
	}
	if target.Environment.SSH.Platform == model.EnvironmentPlatformLinux && !path.IsAbs(root) {
		return nil, nil, nil, errors.New("linux SSH pipeline workspace must resolve to an absolute path")
	}
	root = path.Join(root, workspacepath.PipelineDirName)
	workspace := &pipelineWorkspace{runtime: runtime.ssh, target: target, ctx: ctx, root: root}
	return workspace, &pipelineDockerRunner{runtime: runtime.ssh, target: target, root: root}, &pipelineLogs{workspace: workspace}, nil
}

type pipelineWorkspace struct {
	runtime *Runtime
	target  environmentport.Target
	ctx     context.Context
	root    string
}

func (workspace *pipelineWorkspace) runPath(runId string) (string, error) {
	if !safePathSegment(runId) {
		return "", errors.New("invalid pipeline run ID")
	}
	return path.Join(workspace.root, "runs", runId), nil
}

func (workspace *pipelineWorkspace) CreateRunDirectories(projectCode string, runId string) error {
	if !safePathSegment(projectCode) {
		return errors.New("invalid repository code")
	}
	runPath, err := workspace.runPath(runId)
	if err != nil {
		return err
	}
	client, closeClient, err := workspace.runtime.openSFTP(workspace.ctx, workspace.target)
	if err != nil {
		return err
	}
	defer closeClient()
	for _, directory := range []string{path.Join(workspace.root, projectCode, "workspace"), path.Join(runPath, "artifacts"), path.Join(runPath, "stages")} {
		if err := client.MkdirAll(directory); err != nil {
			return fmt.Errorf("create remote pipeline directory %s: %w", directory, err)
		}
	}
	return nil
}

func (workspace *pipelineWorkspace) ArtifactsPath(runId string) string {
	return path.Join(workspace.root, "runs", runId, "artifacts")
}

func (workspace *pipelineWorkspace) ArtifactLocation(runId string, artifactPath string) (string, error) {
	if _, err := workspace.runPath(runId); err != nil {
		return "", err
	}
	reference := normalizeRemotePath(artifactPath)
	if reference == "." || reference == ".." || strings.HasPrefix(reference, "../") || strings.HasPrefix(reference, "/") || strings.Contains(reference, ":") {
		return "", errors.New("invalid remote artifact path")
	}
	return path.Join(workspace.ArtifactsPath(runId), reference), nil
}

func (workspace *pipelineWorkspace) ArtifactExists(runId string, artifactPath string) (bool, error) {
	location, err := workspace.ArtifactLocation(runId, artifactPath)
	if err != nil {
		return false, err
	}
	client, closeClient, err := workspace.runtime.openSFTP(workspace.ctx, workspace.target)
	if err != nil {
		return false, err
	}
	defer closeClient()
	_, err = client.Stat(location)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func (workspace *pipelineWorkspace) StageLogPath(runId string, stageRunId string) string {
	return path.Join(workspace.root, "runs", runId, "stages", stageRunId+".log")
}

func (workspace *pipelineWorkspace) RemoveRunFiles(runId string) error {
	runPath, err := workspace.runPath(runId)
	if err != nil {
		return err
	}
	client, closeClient, err := workspace.runtime.openSFTP(workspace.ctx, workspace.target)
	if err != nil {
		return err
	}
	defer closeClient()
	if _, err := client.Stat(runPath); err != nil {
		return err
	}
	var files, directories []string
	walker := client.Walk(runPath)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			return err
		}
		entryPath := normalizeRemotePath(walker.Path())
		if entryPath != runPath && !strings.HasPrefix(entryPath, runPath+"/") {
			return errors.New("remote pipeline cleanup escaped run directory")
		}
		if walker.Stat().IsDir() {
			directories = append(directories, entryPath)
		} else {
			files = append(files, entryPath)
		}
	}
	for _, file := range files {
		if err := client.Remove(file); err != nil {
			return fmt.Errorf("remove remote run file: %w", err)
		}
	}
	for index := len(directories) - 1; index >= 0; index-- {
		if err := client.RemoveDirectory(directories[index]); err != nil {
			return fmt.Errorf("remove remote run directory: %w", err)
		}
	}
	return nil
}

func (workspace *pipelineWorkspace) DockerStageMounts(_ context.Context, projectCode string, runId string) ([]pipelinerunport.VolumeMount, error) {
	if !safePathSegment(projectCode) {
		return nil, errors.New("invalid repository code")
	}
	if _, err := workspace.runPath(runId); err != nil {
		return nil, err
	}
	return []pipelinerunport.VolumeMount{
		{HostPath: path.Join(workspace.root, projectCode, "workspace"), ContainerPath: "/workspace", Mode: "rw"},
		{HostPath: workspace.ArtifactsPath(runId), ContainerPath: "/artifacts", Mode: "rw"},
	}, nil
}

type pipelineLogs struct{ workspace *pipelineWorkspace }

func (logs *pipelineLogs) validLogPath(logPath string) bool {
	logPath = normalizeRemotePath(logPath)
	if !strings.HasPrefix(logPath, logs.workspace.root+"/runs/") || path.Ext(logPath) != ".log" {
		return false
	}
	relative := strings.TrimPrefix(logPath, logs.workspace.root+"/runs/")
	parts := strings.Split(relative, "/")
	return len(parts) == 3 && safePathSegment(parts[0]) && parts[1] == "stages" && safePathSegment(strings.TrimSuffix(parts[2], ".log"))
}

func (logs *pipelineLogs) Writer(logPath string) (io.WriteCloser, error) {
	if !logs.validLogPath(logPath) {
		return nil, errors.New("invalid remote stage log path")
	}
	client, closeClient, err := logs.workspace.runtime.openSFTP(logs.workspace.ctx, logs.workspace.target)
	if err != nil {
		return nil, err
	}
	file, err := client.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY)
	if err != nil {
		closeClient()
		return nil, fmt.Errorf("open remote stage log: %w", err)
	}
	if logs.workspace.target.Environment.SSH.Platform == model.EnvironmentPlatformWindows {
		if err := file.Close(); err != nil {
			closeClient()
			return nil, fmt.Errorf("close remote stage log: %w", err)
		}
		return &remoteLogWriter{client: client, path: logPath, ctx: logs.workspace.ctx, closeClient: closeClient}, nil
	}
	return &remoteLogWriter{file: file, closeClient: closeClient}, nil
}

func (logs *pipelineLogs) Read(logPath string, offset int) ([]byte, int, error) {
	if !logs.validLogPath(logPath) || offset < 0 {
		return nil, offset, errors.New("invalid remote stage log read")
	}
	client, closeClient, err := logs.workspace.runtime.openSFTP(logs.workspace.ctx, logs.workspace.target)
	if err != nil {
		return nil, offset, err
	}
	defer closeClient()
	file, err := openStageLogFile(logs.workspace.ctx, logs.workspace.target.Environment.SSH.Platform, func() (*sftp.File, error) {
		return client.Open(logPath)
	})
	if os.IsNotExist(err) {
		return nil, offset, nil
	}
	if err != nil {
		return nil, offset, fmt.Errorf("open remote stage log: %w", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return nil, offset, fmt.Errorf("stat remote stage log: %w", err)
	}
	return readRemoteStageLog(file, info.Size(), offset)
}

func openStageLogFile(ctx context.Context, platform string, open func() (*sftp.File, error)) (*sftp.File, error) {
	file, err := open()
	for attempts := 0; attempts < 40 && platform == model.EnvironmentPlatformWindows && isSFTPSharingFailure(err); attempts++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
		file, err = open()
	}
	return file, err
}

func isSFTPSharingFailure(err error) bool {
	var statusError *sftp.StatusError
	return errors.As(err, &statusError) && statusError.FxCode() == sftp.ErrSSHFxFailure
}

func readRemoteStageLog(file io.ReadSeeker, size int64, offset int) ([]byte, int, error) {
	if int64(offset) >= size {
		return nil, offset, nil
	}
	if _, err := file.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, offset, fmt.Errorf("seek remote stage log: %w", err)
	}
	content, err := io.ReadAll(io.LimitReader(file, size-int64(offset)))
	if err != nil {
		return content, offset + len(content), fmt.Errorf("read remote stage log: %w", err)
	}
	return content, offset + len(content), nil
}

type remoteLogWriter struct {
	mu          sync.Mutex
	file        *sftp.File
	client      *sftp.Client
	path        string
	ctx         context.Context
	closeClient func()
	closed      bool
}

func (writer *remoteLogWriter) Write(content []byte) (int, error) {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.closed {
		return 0, os.ErrClosed
	}
	if writer.file != nil {
		return writer.file.Write(content)
	}
	if len(content) == 0 {
		return 0, nil
	}
	file, err := openStageLogFile(writer.ctx, model.EnvironmentPlatformWindows, func() (*sftp.File, error) {
		return writer.client.OpenFile(writer.path, os.O_APPEND|os.O_WRONLY)
	})
	if err != nil {
		return 0, fmt.Errorf("open remote stage log append: %w", err)
	}
	count, writeErr := file.Write(content)
	closeErr := file.Close()
	if writeErr != nil {
		return count, fmt.Errorf("write remote stage log: %w", errors.Join(writeErr, closeErr))
	}
	if closeErr != nil {
		return count, fmt.Errorf("close remote stage log append: %w", closeErr)
	}
	return count, nil
}

func (writer *remoteLogWriter) Close() error {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.closed {
		return nil
	}
	writer.closed = true
	var err error
	if writer.file != nil {
		err = writer.file.Close()
	}
	writer.closeClient()
	return err
}

type pipelineDockerRunner struct {
	runtime *Runtime
	target  environmentport.Target
	root    string
}

func (runner *pipelineDockerRunner) dockerArgs(opts pipelinerunport.RunOptions, containerName string) ([]string, error) {
	if strings.TrimSpace(opts.Image) == "" || !safePathSegment(containerName) {
		return nil, errors.New("invalid remote stage image or container name")
	}
	args := []string{"run", "--rm", "-i", "-w", "/workspace", "--entrypoint", "sh", "--name", containerName}
	for _, value := range opts.Environment {
		args = append(args, "-e", value)
	}
	for _, volume := range opts.Volumes {
		host := normalizeRemotePath(volume.HostPath)
		validHost := windowsDrivePath.MatchString(host)
		if runner.target.Environment.SSH.Platform == model.EnvironmentPlatformLinux {
			validHost = path.IsAbs(host)
		}
		if !validHost || strings.Contains(host, ",") || !strings.HasPrefix(volume.ContainerPath, "/") || strings.Contains(volume.ContainerPath, ",") {
			return nil, errors.New("invalid remote Docker bind mount")
		}
		mount := "type=bind,source=" + host + ",target=" + volume.ContainerPath
		if volume.Mode == "ro" {
			mount += ",readonly"
		}
		args = append(args, "--mount", mount)
	}
	args = append(args, "--mount", "type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock", opts.Image, "-x", "-s")
	return args, nil
}

func (runner *pipelineDockerRunner) runDocker(ctx context.Context, args []string, script string, stdout io.Writer, stderr io.Writer) error {
	command, _, err := remoteCommand(runner.target.Environment.SSH.Platform, runner.root, "docker", args...)
	if err != nil {
		return err
	}
	client, closeClient, err := runner.runtime.openSSH(ctx, runner.target)
	if err != nil {
		return err
	}
	defer closeClient()
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create pipeline SSH session: %w", err)
	}
	defer func() { _ = session.Close() }()
	session.Stdin = strings.NewReader(script)
	session.Stdout, session.Stderr = stdout, stderr
	return session.Run(command)
}

func (runner *pipelineDockerRunner) Run(ctx context.Context, opts pipelinerunport.RunOptions) (int, string, error) {
	args, err := runner.dockerArgs(opts, opts.ContainerName)
	if err != nil {
		return 1, "", err
	}
	err = runner.runDocker(ctx, args, opts.Script, opts.LogWriter, opts.LogWriter)
	if ctx.Err() != nil {
		cleanupErr := runner.removeContainer(opts.ContainerName)
		if cleanupErr != nil {
			return 1, "", fmt.Errorf("%w; remove remote container: %v", ctx.Err(), cleanupErr)
		}
		return 1, "", ctx.Err()
	}
	if err == nil {
		return 0, "", nil
	}
	var exitError *ssh.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitStatus(), "", nil
	}
	return 1, "", fmt.Errorf("run SSH pipeline container: %w", err)
}

func (runner *pipelineDockerRunner) RunCommand(ctx context.Context, opts pipelinerunport.RunOptions, command string) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "", errors.New("command artifact collector requires a command")
	}
	containerName := opts.ContainerName + "-collector"
	args, err := runner.dockerArgs(opts, containerName)
	if err != nil {
		return "", err
	}
	var stdout, stderr bytes.Buffer
	err = runner.runDocker(ctx, args, command, &stdout, &stderr)
	if ctx.Err() != nil {
		if cleanupErr := runner.removeContainer(containerName); cleanupErr != nil {
			return "", fmt.Errorf("%w; remove remote collector: %v", ctx.Err(), cleanupErr)
		}
		return "", ctx.Err()
	}
	if err != nil {
		return "", fmt.Errorf("run SSH command artifact: %w: %s", err, boundedRemoteOutput(stderr.String()))
	}
	return stdout.String(), nil
}

func (runner *pipelineDockerRunner) ImageId(ctx context.Context, imageRef string) (string, error) {
	if strings.TrimSpace(imageRef) == "" {
		return "", errors.New("image reference is required")
	}
	command, _, err := remoteCommand(runner.target.Environment.SSH.Platform, runner.root, "docker", "image", "inspect", "--format", "{{.Id}}", imageRef)
	if err != nil {
		return "", err
	}
	client, closeClient, err := runner.runtime.openSSH(ctx, runner.target)
	if err != nil {
		return "", err
	}
	defer closeClient()
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer func() { _ = session.Close() }()
	var stdout, stderr bytes.Buffer
	session.Stdout, session.Stderr = &stdout, &stderr
	if err := session.Run(command); err != nil {
		return "", fmt.Errorf("inspect remote image: %w: %s", err, boundedRemoteOutput(stderr.String()))
	}
	imageId := strings.TrimSpace(stdout.String())
	if !strings.HasPrefix(imageId, "sha256:") || len(imageId) <= len("sha256:") {
		return "", errors.New("remote image inspect returned invalid image ID")
	}
	return imageId, nil
}

func (runner *pipelineDockerRunner) removeContainer(containerName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command, _, err := remoteCommand(runner.target.Environment.SSH.Platform, runner.root, "docker", "rm", "--force", containerName)
	if err != nil {
		return err
	}
	client, closeClient, err := runner.runtime.openSSH(ctx, runner.target)
	if err != nil {
		return err
	}
	defer closeClient()
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()
	var output bytes.Buffer
	session.Stdout, session.Stderr = &output, &output
	if err := session.Run(command); err != nil && !strings.Contains(output.String(), "No such container") {
		return fmt.Errorf("clean up remote stage container: %w: %s", err, boundedRemoteOutput(output.String()))
	}
	return nil
}
