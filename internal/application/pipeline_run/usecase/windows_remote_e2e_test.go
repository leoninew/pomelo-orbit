package pipelinerunsvc

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
	pipelinevariable "github.com/leoninew/pomelo-orbit/internal/application/pipeline/rule/pipelinevariable"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	sshrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/ssh"
	"github.com/leoninew/pomelo-orbit/internal/model"
	environmentrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/environment"
	environmentcredentialrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/environment_credential"
)

func TestWindowsPipelineApplicationRemoteEndToEnd(t *testing.T) {
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
		t.Fatal("selected Project is not a Windows SSH target")
	}
	workspace, container, logs, err := sshrunner.NewPipelineRuntime().RuntimeForTarget(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	runId := "windows-app-e2e-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		cleanupWorkspace, _, _, err := sshrunner.NewPipelineRuntime().RuntimeForTarget(cleanupCtx, target)
		if err != nil {
			t.Errorf("open remote cleanup: %v", err)
			return
		}
		if err := cleanupWorkspace.RemoveRunFiles(runId); err != nil && !os.IsNotExist(err) {
			t.Errorf("remove remote run: %v", err)
		}
	})
	if err := workspace.CreateRunDirectories("windows-e2e", runId); err != nil {
		t.Fatal(err)
	}
	run := model.PipelineRun{Id: runId, ProjectId: &projectId, RepositoryId: "repository", RepositoryName: "test",
		PipelineId: "pipeline", PipelineName: "Windows E2E"}
	stages := []model.StageDefinition{
		{Id: "build", Name: "build", Image: image, Script: "echo remote-report > /artifacts/report.txt; echo build-complete", Artifacts: []model.ArtifactConfig{
			{Name: "report", Collector: "file", Reference: "report.txt"},
			{Name: "command", Collector: "command", Command: "printf collector-ready", Format: "text"},
			{Name: "image", Collector: "docker_image", Reference: image},
		}},
		{Id: "verify", Name: "verify", Image: image, Script: "test -f /artifacts/report.txt; echo verified", DependsOn: []string{"build"}},
	}
	stageRuns := map[string]model.PipelineStageRun{
		"build":  {Id: "stage-build", StageId: "build", StageName: "build", PipelineRunId: runId},
		"verify": {Id: "stage-verify", StageId: "verify", StageName: "verify", PipelineRunId: runId},
	}
	store := &remoteExecutionStore{}
	executor := Executor{store: store, workspace: workspace, logStore: logs, runner: container,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)), executionTimeout: 3 * time.Minute}
	ok, message := executor.Execute(ctx, ctx, projectId, run,
		model.Repository{Id: "repository", Code: "windows-e2e", RepositoryType: model.RepositoryTypeRemoteGit,
			RepositoryUrl: "https://git.example.test/repository.git"},
		pipelinevariable.RuntimeVariables{Global: map[string]any{}}, stages, stageRuns)
	if !ok {
		t.Fatalf("remote DAG failed: %s", message)
	}
	if store.statuses["build"] != status.WorkStatusRanToCompletion || store.statuses["verify"] != status.WorkStatusRanToCompletion {
		t.Fatalf("remote stages = %v", store.statuses)
	}
	if len(store.artifacts) != 3 || store.artifacts[0].Location == nil ||
		!strings.HasPrefix(*store.artifacts[0].Location, workspace.ArtifactsPath(runId)) ||
		store.artifacts[1].Value == nil || *store.artifacts[1].Value != "collector-ready" ||
		store.artifacts[2].LocalImageSha256 == nil || !strings.HasPrefix(*store.artifacts[2].LocalImageSha256, "sha256:") {
		t.Fatalf("remote artifacts = %+v", store.artifacts)
	}
	content, _, err := logs.Read(workspace.StageLogPath(runId, "stage-verify"), 0)
	if err != nil || !strings.Contains(string(content), "verified") {
		t.Fatalf("remote DAG log = %q, err=%v", content, err)
	}
}
