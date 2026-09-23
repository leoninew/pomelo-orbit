package sshrunner

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
	pipelinerunport "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/port"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
	environmentrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/environment"
	environmentcredentialrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/environment_credential"
)

func TestWindowsPipelineRemoteEndToEnd(t *testing.T) {
	projectId := os.Getenv("POMELO_ORBIT_WINDOWS_PIPELINE_E2E_PROJECT_ID")
	image := os.Getenv("POMELO_ORBIT_WINDOWS_PIPELINE_E2E_IMAGE")
	if projectId == "" || image == "" {
		t.Skip("set Windows SSH Project ID and an already-available shell image to run remote E2E")
	}
	_, sourceFile, _, _ := runtime.Caller(0)
	originalDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Join(filepath.Dir(sourceFile), "../../../..")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDirectory) })
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	database, err := db.Open(cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	target, err := environmentsvc.NewTargetResolver(environmentrepo.NewRepository(database),
		environmentcredentialrepo.NewRepository(database), cfg.Jwt.SecretKey).ResolveProjectTarget(ctx, projectId)
	if err != nil {
		t.Fatal(err)
	}
	if !target.Environment.IsSSH() || target.Environment.SSH.Platform != model.EnvironmentPlatformWindows {
		t.Fatal("selected Project is not a Windows SSH pipeline target")
	}
	untrustedTarget := target
	sshTarget := *target.Environment.SSH
	untrustedTarget.Environment.SSH = &sshTarget
	untrustedTarget.Environment.SSH.HostKeyFingerprint = "SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	if _, _, _, err := NewPipelineRuntime().RuntimeForTarget(ctx, untrustedTarget); err == nil || !strings.Contains(err.Error(), "host key fingerprint mismatch") {
		t.Fatalf("remote target accepted an untrusted host key: %v", err)
	}
	workspace, container, logs, err := NewPipelineRuntime().RuntimeForTarget(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	runner := container.(*pipelineDockerRunner)
	runId := "windows-e2e-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	stageId := "stage-1"
	t.Cleanup(func() {
		cleanupCtx, stopCleanup := context.WithTimeout(context.Background(), 30*time.Second)
		defer stopCleanup()
		cleanupWorkspace := *workspace.(*pipelineWorkspace)
		cleanupWorkspace.ctx = cleanupCtx
		if err := cleanupWorkspace.RemoveRunFiles(runId); err != nil && !os.IsNotExist(err) {
			t.Errorf("cleanup remote run: %v", err)
		}
	})
	if err := workspace.CreateRunDirectories("windows-e2e", runId); err != nil {
		t.Fatal(err)
	}
	mounts, err := workspace.DockerStageMounts(ctx, "windows-e2e", runId)
	if err != nil {
		t.Fatal(err)
	}
	options := pipelinerunport.RunOptions{ContainerName: "pomelo-orbit-stage-" + runId,
		Image: image, Script: "echo remote-stage-ready; echo remote-artifact > /artifacts/report.txt", Volumes: mounts}
	logPath := workspace.StageLogPath(runId, stageId)
	writer, err := logs.Writer(logPath)
	if err != nil {
		t.Fatal(err)
	}
	options.LogWriter = writer
	exitCode, _, runErr := runner.Run(ctx, options)
	if closeErr := writer.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if runErr != nil || exitCode != 0 {
		content, _, _ := logs.Read(logPath, 0)
		t.Fatalf("remote stage exit=%d err=%v log=%s", exitCode, runErr, boundedRemoteOutput(string(content)))
	}
	content, next, err := logs.Read(logPath, 0)
	if err != nil || !bytes.Contains(content, []byte("remote-stage-ready")) || next != len(content) {
		t.Fatalf("remote log read offset=%d err=%v content=%q", next, err, content)
	}
	more, later, err := logs.Read(logPath, next)
	if err != nil || len(more) != 0 || later != next {
		t.Fatalf("incremental log read offset=%d err=%v", later, err)
	}
	exists, err := workspace.ArtifactExists(runId, "report.txt")
	if err != nil || !exists {
		t.Fatalf("remote artifact exists=%v err=%v", exists, err)
	}
	location, err := workspace.(pipelinerunport.ArtifactLocator).ArtifactLocation(runId, "report.txt")
	if err != nil || !strings.HasPrefix(location, workspace.ArtifactsPath(runId)) {
		t.Fatalf("remote artifact location=%q err=%v", location, err)
	}
	output, err := runner.RunCommand(ctx, options, "printf collector-ready")
	if err != nil || strings.TrimSpace(output) != "collector-ready" {
		t.Fatalf("remote collector output=%q err=%v", output, err)
	}
	imageId, err := runner.ImageId(ctx, image)
	if err != nil || !strings.HasPrefix(imageId, "sha256:") {
		t.Fatalf("remote image ID=%q err=%v", imageId, err)
	}
	cancelCtx, stop := context.WithTimeout(ctx, 2*time.Second)
	defer stop()
	options.ContainerName += "-cancel"
	options.Script = "sleep 30"
	options.LogWriter = &bytes.Buffer{}
	_, _, err = runner.Run(cancelCtx, options)
	if err == nil || cancelCtx.Err() == nil {
		t.Fatalf("remote cancellation returned %v", err)
	}
	var containerState bytes.Buffer
	if err := runner.runDocker(ctx, []string{"container", "ls", "--all", "--filter", "name=^/" + options.ContainerName + "$", "--format", "{{.Names}}"}, "", &containerState, &containerState); err != nil || strings.TrimSpace(containerState.String()) != "" {
		t.Fatalf("canceled remote container was not removed: %v: %s", err, boundedRemoteOutput(containerState.String()))
	}
	if err := workspace.RemoveRunFiles(runId); err != nil {
		t.Fatal(err)
	}
	exists, err = workspace.ArtifactExists(runId, "report.txt")
	if err != nil || exists {
		t.Fatalf("remote artifact after cleanup exists=%v err=%v", exists, err)
	}
	t.Logf("Windows SSH pipeline probe and runtime verified for Project %s", projectId)
}
