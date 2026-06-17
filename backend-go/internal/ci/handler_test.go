package ci

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"backend/internal/config"
	"backend/internal/repository"
	"backend/internal/repository/model"
)

func TestEngineMarksRunFaultedWhenStageFails(t *testing.T) {
	store := &fakeStore{
		run:  model.PipelineRun{Id: "run-1", RepositoryId: "repo-1", SnapshotId: "snapshot-1", VariablesSnapshot: `{}`},
		repo: model.Repository{Id: "repo-1", Code: "repo"},
		snapshot: model.PipelineSnapshot{Id: "snapshot-1", StagesSnapshot: `[
			{"id":"stage-1","name":"build","image":"alpine","script":"exit 1"},
			{"id":"stage-2","name":"deploy","image":"alpine","depends_on":["stage-1"],"script":"echo deploy"}
		]`},
	}
	engine := Engine{store: store, cfg: config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, logger: slog.Default(), runner: failingContainerRunner{}}

	err := engine.Execute(context.Background(), ExecuteInput{PipelineRunId: "run-1"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if store.runStatus != repository.WorkStatusFaulted {
		t.Fatalf("unexpected final run status: %s", store.runStatus)
	}
	if len(store.stageRuns) != 2 {
		t.Fatalf("expected failed and canceled stage runs, got %d", len(store.stageRuns))
	}
	var canceled bool
	for _, stageRun := range store.stageRuns {
		if stageRun.Status == repository.WorkStatusCanceled {
			canceled = true
		}
	}
	if !canceled {
		t.Fatalf("expected one stage canceled, got %+v", store.stageRuns)
	}
}

func TestEngineExecutesPipelineRun(t *testing.T) {
	store := &fakeStore{
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
	engine := Engine{
		store:  store,
		cfg:    config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}},
		logger: slog.Default(),
		runner: fakeContainerRunner{},
	}

	err := engine.Execute(context.Background(), ExecuteInput{PipelineRunId: "run-1"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !store.runStarted {
		t.Fatal("expected run to be marked running")
	}
	if store.runStatus != repository.WorkStatusRanToCompletion {
		t.Fatalf("unexpected final run status: %s", store.runStatus)
	}
	if len(store.stageRuns) != 1 {
		t.Fatalf("expected 1 stage run, got %d", len(store.stageRuns))
	}
	if store.stageRuns[0].Status != repository.WorkStatusRanToCompletion {
		t.Fatalf("unexpected stage status: %s", store.stageRuns[0].Status)
	}
}

type fakeStore struct {
	mu         sync.Mutex
	run        model.PipelineRun
	repo       model.Repository
	snapshot   model.PipelineSnapshot
	stageRuns  []model.StageRun
	runStarted bool
	runStatus  string
}

func (s *fakeStore) PipelineRun(ctx context.Context, id string) (model.PipelineRun, error) {
	return s.run, nil
}

func (s *fakeStore) Repository(ctx context.Context, id string) (model.Repository, error) {
	return s.repo, nil
}

func (s *fakeStore) PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error) {
	return s.snapshot, nil
}

func (s *fakeStore) MarkPipelineRunRunning(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runStarted = true
	return nil
}

func (s *fakeStore) CompletePipelineRun(ctx context.Context, id string, status string, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runStatus = status
	return nil
}

func (s *fakeStore) InsertStageRun(ctx context.Context, stage model.StageRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stageRuns = append(s.stageRuns, stage)
	return nil
}

func (s *fakeStore) UpdateStageRun(ctx context.Context, stage model.StageRun) error {
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

func (s *fakeStore) InsertArtifact(ctx context.Context, projectId *string, run model.PipelineRun, stageName string, artifact model.ArtifactConfig, path string) error {
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
