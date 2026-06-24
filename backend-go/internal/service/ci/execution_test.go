package cisvc

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"backend/internal/repository/model"
	"backend/internal/security"
	"backend/internal/status"
)

const testExecutionFernetKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func TestExecutePipelineRunMarksRunFaultedWhenStageFails(t *testing.T) {
	store := &fakeExecutionStore{
		run:  model.PipelineRun{Id: "run-1", RepositoryId: "repo-1", SnapshotId: "snapshot-1", VariablesSnapshot: `{}`},
		repo: model.Repository{Id: "repo-1", Code: "repo"},
		snapshot: model.PipelineSnapshot{Id: "snapshot-1", StagesSnapshot: `[
			{"id":"stage-1","name":"build","image":"alpine","script":"exit 1"},
			{"id":"stage-2","name":"deploy","image":"alpine","depends_on":["stage-1"],"script":"echo deploy"}
		]`},
	}
	service := NewExecutionService(store, t.TempDir(), testExecutionFernetKey, slog.Default(), failingContainerRunner{})

	err := service.ExecutePipelineRun(context.Background(), ExecutePipelineRunInput{PipelineRunId: "run-1"})
	if err != nil {
		t.Fatalf("ExecutePipelineRun returned error: %v", err)
	}
	if store.runStatus != status.WorkStatusFaulted {
		t.Fatalf("unexpected final run status: %s", store.runStatus)
	}
	if len(store.stageRuns) != 2 {
		t.Fatalf("expected failed and canceled stage runs, got %d", len(store.stageRuns))
	}
	var canceled bool
	for _, stageRun := range store.stageRuns {
		if stageRun.Status == status.WorkStatusCanceled {
			canceled = true
		}
	}
	if !canceled {
		t.Fatalf("expected one stage canceled, got %+v", store.stageRuns)
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
			VariablesSnapshot: `{}`,
		},
		repo:     model.Repository{Id: "repo-1", Code: "repo"},
		snapshot: model.PipelineSnapshot{Id: "snapshot-1", StagesSnapshot: `[{"id":"stage-1","name":"build","image":"alpine","script":"echo ok"}]`},
	}
	service := NewExecutionService(store, t.TempDir(), testExecutionFernetKey, slog.Default(), fakeContainerRunner{})

	err := service.ExecutePipelineRun(context.Background(), ExecutePipelineRunInput{PipelineRunId: "run-1"})
	if err != nil {
		t.Fatalf("ExecutePipelineRun returned error: %v", err)
	}
	if !store.runStarted {
		t.Fatal("expected run to be marked running")
	}
	if store.runStatus != status.WorkStatusRanToCompletion {
		t.Fatalf("unexpected final run status: %s", store.runStatus)
	}
	if len(store.stageRuns) != 1 {
		t.Fatalf("expected 1 stage run, got %d", len(store.stageRuns))
	}
	if store.stageRuns[0].Status != status.WorkStatusRanToCompletion {
		t.Fatalf("unexpected stage status: %s", store.stageRuns[0].Status)
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
		repo:       model.Repository{Id: "repo-1", Name: "Repo", Code: "repo", RepositoryURL: "https://gitee.com/leoninew/pomelo-orbit.git", GitCredentialId: &credentialId},
		credential: model.Credential{Id: credentialId, Type: "gitee_token", EncryptedData: encrypted},
		template:   model.PipelineTemplate{Id: "template-1", Name: "template", Version: 1},
		snapshot:   model.PipelineSnapshot{Id: "snapshot-1", StagesSnapshot: `[{"id":"stage-1","name":"git clone","image":"alpine/git","script":"git remote add origin {{ repository_url }}\ngit fetch --depth=1 origin {{ repository_ref }}"}]`, VariablesSnapshot: `[{"name":"repository_url","source":"template","editable":false},{"name":"repository_ref","source":"template","editable":false}]`},
	}
	runner := &recordingContainerRunner{}
	service := NewExecutionService(store, t.TempDir(), testExecutionFernetKey, slog.Default(), runner)

	if err := service.ExecutePipelineRun(context.Background(), ExecutePipelineRunInput{PipelineRunId: "run-1"}); err != nil {
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
		repo:     model.Repository{Id: "repo-1", Name: "Repo", Code: "repo", RepositoryURL: "https://example.test/repo.git"},
		template: model.PipelineTemplate{Id: "template-1", Name: "template", Version: 1},
		snapshot: model.PipelineSnapshot{Id: "snapshot-1", StagesSnapshot: `[{"id":"stage-1","name":"build","image":"alpine","script":"cd {{ working_dir }} && echo {{ repository_code }}"}]`, VariablesSnapshot: `[{"name":"working_dir","default":".","source":"template_stage","editable":true},{"name":"repository_code","source":"template","editable":false}]`},
	}
	runner := &recordingContainerRunner{}
	service := NewExecutionService(store, t.TempDir(), testExecutionFernetKey, slog.Default(), runner)

	if err := service.ExecutePipelineRun(context.Background(), ExecutePipelineRunInput{PipelineRunId: "run-1", Variables: map[string]any{"working_dir": "ignored"}}); err != nil {
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
	mu         sync.Mutex
	run        model.PipelineRun
	repo       model.Repository
	credential model.Credential
	snapshot   model.PipelineSnapshot
	template   model.PipelineTemplate
	stageRuns  []model.StageRun
	runStarted bool
	runStatus  string
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

func (s *fakeExecutionStore) InsertStageRun(ctx context.Context, stage model.StageRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stageRuns = append(s.stageRuns, stage)
	return nil
}

func (s *fakeExecutionStore) UpdateStageRun(ctx context.Context, stage model.StageRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.stageRuns {
		if s.stageRuns[i].Id == stage.Id {
			s.stageRuns[i] = stage
			return nil
		}
	}
	s.stageRuns = append(s.stageRuns, stage)
	return nil
}

func (s *fakeExecutionStore) InsertArtifact(ctx context.Context, projectId *string, run model.PipelineRun, stageName string, artifact model.ArtifactConfig, path string) error {
	return nil
}

type fakeContainerRunner struct{}

func (fakeContainerRunner) Run(ctx context.Context, opts RunOptions) (int, string, error) {
	if opts.LogFile != nil {
		_, _ = opts.LogFile.Write([]byte("ok"))
	}
	return 0, "ok", nil
}

type failingContainerRunner struct{}

func (failingContainerRunner) Run(ctx context.Context, opts RunOptions) (int, string, error) {
	return 1, "line1\nline2\nline3\nline4", nil
}

type recordingContainerRunner struct {
	script      string
	environment []string
	volumes     []VolumeMount
}

func (r *recordingContainerRunner) Run(ctx context.Context, opts RunOptions) (int, string, error) {
	r.script = opts.Script
	r.environment = opts.Environment
	r.volumes = opts.Volumes
	return 0, "ok", nil
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
