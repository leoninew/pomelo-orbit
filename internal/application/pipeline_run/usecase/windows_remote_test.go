package pipelinerunsvc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	pipelinevariable "github.com/leoninew/pomelo-orbit/internal/application/pipeline/rule/pipelinevariable"
	pipelinerunport "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type remoteWorkspaceStub struct {
	pipelinerunport.Workspace
	removeError error
	removed     string
}

func (workspace *remoteWorkspaceStub) CreateRunDirectories(string, string) error { return nil }
func (workspace *remoteWorkspaceStub) ArtifactsPath(runId string) string {
	return "D:/remote/pipeline/runs/" + runId + "/artifacts"
}
func (workspace *remoteWorkspaceStub) ArtifactExists(string, string) (bool, error) { return true, nil }
func (workspace *remoteWorkspaceStub) ArtifactLocation(runId string, reference string) (string, error) {
	return workspace.ArtifactsPath(runId) + "/" + reference, nil
}
func (workspace *remoteWorkspaceStub) StageLogPath(runId string, stageId string) string {
	return "D:/remote/pipeline/runs/" + runId + "/stages/" + stageId + ".log"
}
func (workspace *remoteWorkspaceStub) RemoveRunFiles(runId string) error {
	workspace.removed = runId
	return workspace.removeError
}
func (workspace *remoteWorkspaceStub) DockerStageMounts(context.Context, string, string) ([]pipelinerunport.VolumeMount, error) {
	return []pipelinerunport.VolumeMount{{HostPath: "D:/remote/pipeline/repository/workspace", ContainerPath: "/workspace"}}, nil
}

type remoteLogStub struct{ content map[string]*bytes.Buffer }

type logWriteCloser struct{ io.Writer }

func (logWriteCloser) Close() error { return nil }
func (logs *remoteLogStub) Writer(path string) (io.WriteCloser, error) {
	if logs.content == nil {
		logs.content = map[string]*bytes.Buffer{}
	}
	logs.content[path] = &bytes.Buffer{}
	return logWriteCloser{logs.content[path]}, nil
}
func (logs *remoteLogStub) Read(path string, offset int) ([]byte, int, error) {
	if logs.content[path] == nil {
		return nil, offset, nil
	}
	content := logs.content[path].Bytes()
	if offset > len(content) {
		return nil, offset, nil
	}
	return content[offset:], len(content), nil
}

type remoteRunnerStub struct {
	stages   []string
	exitCode int
}

func (runner *remoteRunnerStub) Run(_ context.Context, options pipelinerunport.RunOptions) (int, string, error) {
	runner.stages = append(runner.stages, options.ContainerName)
	_, _ = io.WriteString(options.LogWriter, "remote stage log\n")
	return runner.exitCode, "", nil
}
func (*remoteRunnerStub) RunCommand(context.Context, pipelinerunport.RunOptions, string) (string, error) {
	return "collected", nil
}
func (*remoteRunnerStub) ImageId(context.Context, string) (string, error) {
	return "sha256:remote-image", nil
}

type remoteResourcesStub struct {
	workspace *remoteWorkspaceStub
	runner    *remoteRunnerStub
	logs      *remoteLogStub
}

func (resources remoteResourcesStub) RuntimeForTarget(_ context.Context, target environmentport.Target) (pipelinerunport.Workspace, pipelinerunport.ContainerRunner, pipelinerunport.ExecutionLogStore, error) {
	if target.Environment.SSH.Platform != model.EnvironmentPlatformWindows {
		return nil, nil, nil, errors.New("unexpected platform")
	}
	return resources.workspace, resources.runner, resources.logs, nil
}

type remoteExecutionStore struct {
	pipelineExecutionStore
	artifacts []model.Artifact
	statuses  map[string]string
}

func (*remoteExecutionStore) BeginPipelineStageRun(context.Context, string, string) (bool, error) {
	return true, nil
}
func (store *remoteExecutionStore) CompletePipelineStageRun(_ context.Context, _ string, stage model.PipelineStageRun) (bool, error) {
	if store.statuses == nil {
		store.statuses = map[string]string{}
	}
	store.statuses[stage.StageId] = stage.Status
	return true, nil
}
func (store *remoteExecutionStore) CreateArtifact(_ context.Context, artifact model.Artifact) error {
	store.artifacts = append(store.artifacts, artifact)
	return nil
}
func (*remoteExecutionStore) PipelineRunVersionBinding(context.Context, string, string) (model.PipelineRunVersionBinding, error) {
	return model.PipelineRunVersionBinding{}, repository.ErrNotFound
}

func TestWindowsRuntimeExecutesDAGAndStoresRemoteArtifacts(t *testing.T) {
	workspace := &remoteWorkspaceStub{}
	logs := &remoteLogStub{}
	runner := &remoteRunnerStub{}
	store := &remoteExecutionStore{}
	projectId := "project-1"
	run := model.PipelineRun{Id: "run-1", ProjectId: &projectId, RepositoryId: "repo-1", RepositoryName: "repo",
		PipelineId: "pipeline-1", PipelineName: "pipeline"}
	stages := []model.StageDefinition{
		{Id: "stage-1", Name: "build", Image: "demo:build", Script: "echo build", Artifacts: []model.ArtifactConfig{
			{Name: "report", Collector: "file", Reference: "report.txt"},
			{Name: "command", Collector: "command", Command: "printf collected", Format: "text"},
			{Name: "image", Collector: "docker_image", Reference: "demo:build"},
		}},
		{Id: "stage-2", Name: "check", Image: "demo:build", Script: "echo check", DependsOn: []string{"stage-1"}},
	}
	stageRuns := map[string]model.PipelineStageRun{
		"stage-1": {Id: "stage-run-1", StageId: "stage-1", StageName: "build", PipelineRunId: run.Id},
		"stage-2": {Id: "stage-run-2", StageId: "stage-2", StageName: "check", PipelineRunId: run.Id},
	}
	executor := Executor{store: store, workspace: workspace, logStore: logs, runner: runner, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	ok, message := executor.Execute(context.Background(), context.Background(), projectId, run,
		model.Repository{Id: "repo-1", Code: "repository", RepositoryType: model.RepositoryTypeRemoteGit},
		pipelinevariable.RuntimeVariables{Global: map[string]any{}}, stages, stageRuns)
	if !ok || message != "" {
		t.Fatalf("remote DAG execution ok=%v message=%q", ok, message)
	}
	if len(runner.stages) != 2 || !strings.HasSuffix(runner.stages[0], "stage-run-1") || !strings.HasSuffix(runner.stages[1], "stage-run-2") {
		t.Fatalf("remote DAG order = %v", runner.stages)
	}
	if len(store.artifacts) != 3 || store.artifacts[0].Location == nil ||
		*store.artifacts[0].Location != "D:/remote/pipeline/runs/run-1/artifacts/report.txt" ||
		store.artifacts[1].Value == nil || *store.artifacts[1].Value != "collected" ||
		store.artifacts[2].LocalImageSha256 == nil || *store.artifacts[2].LocalImageSha256 != "sha256:remote-image" {
		t.Fatalf("remote artifacts = %+v", store.artifacts)
	}
	if store.statuses["stage-1"] != status.WorkStatusRanToCompletion || store.statuses["stage-2"] != status.WorkStatusRanToCompletion {
		t.Fatalf("remote stages = %v", store.statuses)
	}
	if content, _, _ := logs.Read(workspace.StageLogPath(run.Id, "stage-run-1"), 0); !strings.Contains(string(content), "remote stage log") {
		t.Fatalf("remote stage log = %q", content)
	}
}

func TestWindowsRuntimeStageFailureStopsDependentStage(t *testing.T) {
	workspace, logs, runner := &remoteWorkspaceStub{}, &remoteLogStub{}, &remoteRunnerStub{exitCode: 17}
	store := &remoteExecutionStore{}
	executor := Executor{store: store, workspace: workspace, logStore: logs, runner: runner, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	stages := []model.StageDefinition{{Id: "stage-1", Name: "build", Image: "demo", Script: "false"},
		{Id: "stage-2", Name: "deploy", Image: "demo", DependsOn: []string{"stage-1"}}}
	stageRuns := map[string]model.PipelineStageRun{
		"stage-1": {Id: "stage-run-1", StageId: "stage-1", StageName: "build"},
		"stage-2": {Id: "stage-run-2", StageId: "stage-2", StageName: "deploy"},
	}
	ok, message := executor.Execute(context.Background(), context.Background(), "project-1", model.PipelineRun{Id: "run-1"},
		model.Repository{Code: "repository", RepositoryType: model.RepositoryTypeRemoteGit},
		pipelinevariable.RuntimeVariables{Global: map[string]any{}}, stages, stageRuns)
	if ok || !strings.Contains(message, "build") || len(runner.stages) != 1 || store.statuses["stage-1"] != status.WorkStatusFaulted {
		t.Fatalf("failed remote stage ok=%v message=%q stages=%v statuses=%v", ok, message, runner.stages, store.statuses)
	}
}

func TestWindowsRuntimeSelectionAndSafeRemoteCleanup(t *testing.T) {
	environment := readySSHTarget()
	workspace := &remoteWorkspaceStub{removeError: errors.New("SFTP unavailable")}
	resources := remoteResourcesStub{workspace: workspace, runner: &remoteRunnerStub{}, logs: &remoteLogStub{}}
	target := structTarget(environment)
	service := Service{remoteRuntime: resources, targetResolver: directTriggerTargetResolver{environment: environment},
		environments: &runTargetEnvironmentStore{environment: environment}, workspace: directTriggerWorkspace{}}
	runtime, err := service.runtimeForTarget(context.Background(), target)
	if err != nil || runtime.workspace != workspace || runtime.runner != resources.runner || runtime.reader != resources.logs {
		t.Fatalf("selected remote runtime=%+v err=%v", runtime, err)
	}
	projectId := "project-1"
	run := model.PipelineRun{Id: "run-1", ProjectId: &projectId, Status: status.WorkStatusRanToCompletion, CreatedAt: time.Now()}
	applyRunTargetSnapshot(&run, target)
	record := &pipelineRunDeletionStore{run: run}
	service.store = stores{project: pipelineRunDeletionProjectStore{}, pipelineRun: record}
	if err := service.DeletePipelineRun(context.Background(), "user-1", projectId, run.Id); err == nil || !strings.Contains(err.Error(), "SFTP unavailable") {
		t.Fatalf("cleanup failure error = %v", err)
	}
	if record.deleted {
		t.Fatal("must retain run record when remote cleanup fails")
	}
	workspace.removeError = nil
	if err := service.DeletePipelineRun(context.Background(), "user-1", projectId, run.Id); err != nil {
		t.Fatal(err)
	}
	if !record.deleted || workspace.removed != run.Id {
		t.Fatalf("remote cleanup deleted=%v removed=%q", record.deleted, workspace.removed)
	}
}

func TestWindowsSSHTriggerRetryAndIncrementalLog(t *testing.T) {
	environment := readySSHTarget()
	environmentStore := &runTargetEnvironmentStore{environment: environment}
	repository := model.Repository{Id: "repo-1", Code: "repository", RepositoryType: model.RepositoryTypeRemoteGit,
		RepositoryUrl: "https://git.example.test/repository.git", DefaultBranch: "main"}
	service, runStore := targetTestService(environmentStore, repository)
	workspace, logs, runner := &remoteWorkspaceStub{}, &remoteLogStub{}, &remoteRunnerStub{}
	service.targetResolver = directTriggerTargetResolver{environment: environment}
	service.remoteRuntime = remoteResourcesStub{workspace: workspace, logs: logs, runner: runner}
	first, err := service.TriggerPipeline(context.Background(), "user-1", "project-1", "pipeline-1", "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Run.EnvironmentTargetType == nil || *first.Run.EnvironmentTargetType != model.EnvironmentTargetTypeSSH ||
		first.Run.SSHCredentialId == nil || *first.Run.SSHCredentialId != environment.SSH.CredentialId {
		t.Fatalf("Windows SSH run target = %+v", first.Run)
	}
	runStore.run = first.Run
	environmentStore.environment.TargetRevision++
	environmentStore.environment.LastProbeRevision = &environmentStore.environment.TargetRevision
	service.targetResolver = directTriggerTargetResolver{environment: environmentStore.environment}
	second, err := service.RetryPipelineRun(context.Background(), "user-1", "project-1", first.Run.Id)
	if err != nil {
		t.Fatal(err)
	}
	if second.Run.EnvironmentTargetRevision == nil || *second.Run.EnvironmentTargetRevision != environmentStore.environment.TargetRevision ||
		second.Run.RetryOf == nil || *second.Run.RetryOf != first.Run.Id {
		t.Fatalf("Windows SSH Retry target = %+v", second.Run)
	}
	if _, err := logs.Writer(workspace.StageLogPath(second.Run.Id, "stage-run-1")); err != nil {
		t.Fatal(err)
	}
	logs.content[workspace.StageLogPath(second.Run.Id, "stage-run-1")].WriteString("remote output")
	logStore := runTargetLogStore{directTriggerPipelineRunStore: &directTriggerPipelineRunStore{run: second.Run},
		stageRun: model.PipelineStageRun{Id: "stage-run-1", PipelineRunId: second.Run.Id}}
	service.store.pipelineRun = logStore
	result, err := service.PipelineStageLog(context.Background(), "user-1", "project-1", second.Run.Id, "stage-run-1", 7)
	if err != nil || result.Logs != "output" || result.Offset != len("remote output") {
		t.Fatalf("remote log result = %+v, err = %v", result, err)
	}
}
