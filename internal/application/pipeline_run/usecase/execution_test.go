package pipelinerunsvc

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	pipelinerundto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/dto"
	pipelinerunport "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/port"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	security "gitee.com/leoninew/PomeloOrbit-go/internal/common/crypto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/executionlog"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const testExecutionFernetKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func newTestExecutionService(store pipelineExecutionStore, workspace pipelinerunport.Workspace, secretKey string, logger *slog.Logger, runner pipelinerunport.ContainerRunner, logStore pipelinerunport.ExecutionLogStore) Service {
	return Service{
		executionStore:    store,
		workspace:         workspace,
		logStore:          logStore,
		executionLogStore: logStore,
		secretKey:         secretKey,
		logger:            logger,
		runner:            runner,
	}
}

func TestExecutePipelineRunMarksRunFaultedWhenStageFails(t *testing.T) {
	store := &fakeExecutionStore{
		run:  model.PipelineRun{Id: "run-1", RepositoryId: "repo-1", SnapshotId: "snapshot-1", VariablesSnapshot: `[]`},
		repo: model.Repository{Id: "repo-1", Code: "repo"},
		snapshot: model.PipelineSnapshot{Id: "snapshot-1", StagesSnapshot: `[
			{"id":"stage-1","name":"build","image":"alpine","script":"exit 1"},
			{"id":"stage-2","name":"deploy","image":"alpine","depends_on":["stage-1"],"script":"echo deploy"}
		]`},
	}
	service := newTestExecutionService(store, newTestWorkspace(t), testExecutionFernetKey, slog.Default(), failingContainerRunner{}, executionlog.Store{})

	err := service.ExecutePipelineRun(context.Background(), pipelinerundto.ExecutePipelineRunInput{PipelineRunId: "run-1"})
	if err != nil {
		t.Fatalf("ExecutePipelineRun returned error: %v", err)
	}
	if store.runStatus != status.WorkStatusFaulted {
		t.Fatalf("unexpected final run status: %s", store.runStatus)
	}
	if len(store.pipelineStageRuns) != 2 {
		t.Fatalf("expected failed and canceled pipeline stage runs, got %d", len(store.pipelineStageRuns))
	}
	var canceled bool
	for _, pipelineStageRun := range store.pipelineStageRuns {
		if pipelineStageRun.Status == status.WorkStatusCanceled {
			canceled = true
		}
	}
	if !canceled {
		t.Fatalf("expected one stage canceled, got %+v", store.pipelineStageRuns)
	}
}

func TestExecutePipelineRunExecutesPipelineRun(t *testing.T) {
	store := &fakeExecutionStore{
		run: model.PipelineRun{
			Id:                "run-1",
			RepositoryId:      "repo-1",
			RepositoryName:    "repo",
			SnapshotId:        "snapshot-1",
			TemplateId:        "template-1",
			TemplateName:      "template",
			VariablesSnapshot: `[]`,
		},
		repo:     model.Repository{Id: "repo-1", Code: "repo"},
		snapshot: model.PipelineSnapshot{Id: "snapshot-1", StagesSnapshot: `[{"id":"stage-1","name":"build","image":"alpine","script":"echo ok"}]`},
	}
	service := newTestExecutionService(store, newTestWorkspace(t), testExecutionFernetKey, slog.Default(), fakeContainerRunner{}, executionlog.Store{})

	err := service.ExecutePipelineRun(context.Background(), pipelinerundto.ExecutePipelineRunInput{PipelineRunId: "run-1"})
	if err != nil {
		t.Fatalf("ExecutePipelineRun returned error: %v", err)
	}
	if !store.runStarted {
		t.Fatal("expected run to be marked running")
	}
	if store.runStatus != status.WorkStatusRanToCompletion {
		t.Fatalf("unexpected final run status: %s", store.runStatus)
	}
	if len(store.pipelineStageRuns) != 1 {
		t.Fatalf("expected 1 pipeline stage run, got %d", len(store.pipelineStageRuns))
	}
	if store.pipelineStageRuns[0].Status != status.WorkStatusRanToCompletion {
		t.Fatalf("unexpected stage status: %s", store.pipelineStageRuns[0].Status)
	}
}

func TestExecutePipelineRunInjectsGiteeCredentialRewrite(t *testing.T) {
	credentialId := "credential-1"
	encrypted, err := security.EncryptString(testExecutionFernetKey, "leoninew:gitee-token")
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeExecutionStore{
		run: model.PipelineRun{
			Id:           "run-1",
			RepositoryId: "repo-1",
			SnapshotId:   "snapshot-1",
			TemplateId:   "template-1",
			TriggerRef:   "develop",
		},
		repo:       model.Repository{Id: "repo-1", Name: "Repo", Code: "repo", RepositoryUrl: "https://gitee.com/leoninew/pomelo-orbit.git", GitCredentialId: &credentialId},
		credential: model.Credential{Id: credentialId, Type: "gitee_token", EncryptedData: encrypted},
		template:   model.PipelineTemplate{Id: "template-1", Name: "template", Version: 1},
		snapshot:   model.PipelineSnapshot{Id: "snapshot-1", StagesSnapshot: `[{"id":"stage-1","name":"git clone","image":"alpine/git","script":"git remote add origin {{ repository_url }}\ngit fetch --depth=1 origin {{ repository_ref }}"}]`, VariablesSnapshot: `[{"name":"repository_url","source":"template","editable":false},{"name":"repository_ref","source":"template","editable":false}]`},
	}
	runner := &recordingContainerRunner{}
	service := newTestExecutionService(store, newTestWorkspace(t), testExecutionFernetKey, slog.Default(), runner, executionlog.Store{})

	if err := service.ExecutePipelineRun(context.Background(), pipelinerundto.ExecutePipelineRunInput{PipelineRunId: "run-1"}); err != nil {
		t.Fatalf("ExecutePipelineRun returned error: %v", err)
	}
	if runner.script != "git remote add origin https://gitee.com/leoninew/pomelo-orbit.git\ngit fetch --depth=1 origin develop" {
		t.Fatalf("expected original clone script to stay unchanged, got script:\n%s", runner.script)
	}
	if strings.Contains(runner.script, "gitee-token") {
		t.Fatalf("expected script not to contain credential data, got script:\n%s", runner.script)
	}
	if !containsString(runner.environment, "GIT_TERMINAL_PROMPT=0") {
		t.Fatalf("expected GIT_TERMINAL_PROMPT=0, got %+v", runner.environment)
	}
	if !containsString(runner.environment, "GIT_CONFIG_COUNT=1") {
		t.Fatalf("expected GIT_CONFIG_COUNT=1, got %+v", runner.environment)
	}
	if !containsString(runner.environment, "GIT_CONFIG_KEY_0=url.https://leoninew:gitee-token@gitee.com/leoninew/pomelo-orbit.git.insteadOf") {
		t.Fatalf("expected authenticated rewrite config key, got %+v", runner.environment)
	}
	if !containsString(runner.environment, "GIT_CONFIG_VALUE_0=https://gitee.com/leoninew/pomelo-orbit.git") {
		t.Fatalf("expected rewrite config value, got %+v", runner.environment)
	}
}

func TestExecutePipelineRunResolvesVariablesFromDeclarations(t *testing.T) {
	store := &fakeExecutionStore{
		run: model.PipelineRun{
			Id:           "run-1",
			RepositoryId: "repo-1",
			SnapshotId:   "snapshot-1",
			TemplateId:   "template-1",
			TriggerRef:   "main",
		},
		repo:     model.Repository{Id: "repo-1", Name: "Repo", Code: "repo", RepositoryUrl: "https://example.test/repo.git"},
		template: model.PipelineTemplate{Id: "template-1", Name: "template", Version: 1},
		snapshot: model.PipelineSnapshot{Id: "snapshot-1", StagesSnapshot: `[{"id":"stage-1","name":"build","image":"alpine","script":"cd {{ working_dir }} && echo {{ repository_code }}"}]`, VariablesSnapshot: `[{"name":"working_dir","default":".","source":"template_stage","editable":true},{"name":"repository_code","source":"template","editable":false}]`},
	}
	runner := &recordingContainerRunner{}
	service := newTestExecutionService(store, newTestWorkspace(t), testExecutionFernetKey, slog.Default(), runner, executionlog.Store{})

	if err := service.ExecutePipelineRun(context.Background(), pipelinerundto.ExecutePipelineRunInput{PipelineRunId: "run-1", Variables: map[string]any{"working_dir": "ignored"}}); err != nil {
		t.Fatalf("ExecutePipelineRun returned error: %v", err)
	}
	if runner.script != "cd . && echo repo" {
		t.Fatalf("unexpected script: %s", runner.script)
	}
	if len(runner.volumes) != 2 {
		t.Fatalf("expected workspace and artifacts mounts, got %+v", runner.volumes)
	}
	for _, volume := range runner.volumes {
		if !filepath.IsAbs(volume.HostPath) {
			t.Fatalf("expected physical absolute host path, got %q", volume.HostPath)
		}
	}
}

type fakeExecutionStore struct {
	mu                sync.Mutex
	run               model.PipelineRun
	repo              model.Repository
	credential        model.Credential
	snapshot          model.PipelineSnapshot
	template          model.PipelineTemplate
	pipelineStageRuns []model.PipelineStageRun
	runStarted        bool
	runStatus         string
}

func (s *fakeExecutionStore) PipelineRun(ctx context.Context, id string) (model.PipelineRun, error) {
	return s.run, nil
}

func (s *fakeExecutionStore) Repository(ctx context.Context, id string) (model.Repository, error) {
	return s.repo, nil
}

func (s *fakeExecutionStore) Credential(ctx context.Context, id string) (model.Credential, error) {
	return s.credential, nil
}

func (s *fakeExecutionStore) PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error) {
	return s.snapshot, nil
}

func (s *fakeExecutionStore) PipelineTemplate(ctx context.Context, id string) (model.PipelineTemplate, error) {
	if s.template.Id != "" {
		return s.template, nil
	}
	return model.PipelineTemplate{Id: id, Name: "template", Version: 1}, nil
}

func (s *fakeExecutionStore) MarkPipelineRunRunning(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runStarted = true
	return nil
}

func (s *fakeExecutionStore) CompletePipelineRun(ctx context.Context, id string, status string, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runStatus = status
	return nil
}

func (s *fakeExecutionStore) InsertPipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pipelineStageRuns = append(s.pipelineStageRuns, stage)
	return nil
}

func (s *fakeExecutionStore) UpdatePipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.pipelineStageRuns {
		if s.pipelineStageRuns[i].Id == stage.Id {
			s.pipelineStageRuns[i] = stage
			return nil
		}
	}
	s.pipelineStageRuns = append(s.pipelineStageRuns, stage)
	return nil
}

func (s *fakeExecutionStore) InsertArtifact(ctx context.Context, projectId *string, run model.PipelineRun, stageName string, artifact model.ArtifactConfig, path string) error {
	return nil
}

type fakeContainerRunner struct{}

func (fakeContainerRunner) Run(ctx context.Context, opts pipelinerunport.RunOptions) (int, string, error) {
	return 0, "ok", nil
}

type failingContainerRunner struct{}

func (failingContainerRunner) Run(ctx context.Context, opts pipelinerunport.RunOptions) (int, string, error) {
	return 1, "line1\nline2\nline3\nline4", nil
}

type recordingContainerRunner struct {
	script      string
	environment []string
	volumes     []pipelinerunport.VolumeMount
}

func (r *recordingContainerRunner) Run(ctx context.Context, opts pipelinerunport.RunOptions) (int, string, error) {
	r.script = opts.Script
	r.environment = opts.Environment
	r.volumes = opts.Volumes
	return 0, "ok", nil
}

type testWorkspace struct {
	root string
}

func newTestWorkspace(t *testing.T) testWorkspace {
	t.Helper()
	return testWorkspace{root: t.TempDir()}
}

func (w testWorkspace) CreateRunDirectories(projectCode string, runId string) error {
	for _, path := range []string{w.workspacePath(projectCode), w.ArtifactsPath(runId)} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (w testWorkspace) ArtifactsPath(runId string) string {
	return filepath.Join(w.root, "pipeline", "runs", runId, "artifacts")
}

func (w testWorkspace) ArtifactExists(runId string, artifactPath string) (bool, error) {
	_, err := os.Stat(filepath.Join(w.ArtifactsPath(runId), artifactPath))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (w testWorkspace) StageLogPath(runId string, pipelineStageRunId string) string {
	return filepath.Join(w.root, "pipeline", "runs", runId, "stages", pipelineStageRunId+".log")
}

func (w testWorkspace) DockerStageMounts(ctx context.Context, projectCode string, runId string) ([]pipelinerunport.VolumeMount, error) {
	return []pipelinerunport.VolumeMount{
		{HostPath: w.workspacePath(projectCode), ContainerPath: "/workspace", Mode: "rw"},
		{HostPath: w.ArtifactsPath(runId), ContainerPath: "/artifacts", Mode: "rw"},
	}, nil
}

func (w testWorkspace) workspacePath(projectCode string) string {
	return filepath.Join(w.root, "pipeline", projectCode, "workspace")
}

func containsString(values []string, expected string) bool {
	return slices.Contains(values, expected)
}
