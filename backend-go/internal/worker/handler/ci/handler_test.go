package ci

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"backend/internal/config"
	"backend/internal/repository/model"
	taskrepo "backend/internal/repository/task"
)

func TestHandleRejectsInvalidPayload(t *testing.T) {
	handler := NewHandler(&fakeStore{}, config.Config{}, slog.Default())
	err := handler.Handle(context.Background(), taskrepo.Task{PayloadJSON: `{invalid`})
	if err == nil || !strings.Contains(err.Error(), "parse ci task payload") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestHandleRequiresPipelineRunId(t *testing.T) {
	handler := NewHandler(&fakeStore{}, config.Config{}, slog.Default())
	err := handler.Handle(context.Background(), taskrepo.Task{PayloadJSON: `{}`})
	if err == nil || !strings.Contains(err.Error(), "pipeline_run_id is required") {
		t.Fatalf("expected required field error, got %v", err)
	}
}

type fakeStore struct{}

func (s *fakeStore) PipelineRun(ctx context.Context, id string) (model.PipelineRun, error) {
	return model.PipelineRun{}, nil
}

func (s *fakeStore) Repository(ctx context.Context, id string) (model.Repository, error) {
	return model.Repository{}, nil
}

func (s *fakeStore) PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error) {
	return model.PipelineSnapshot{}, nil
}

func (s *fakeStore) MarkPipelineRunRunning(ctx context.Context, id string) error {
	return nil
}

func (s *fakeStore) CompletePipelineRun(ctx context.Context, id string, status string, message string) error {
	return nil
}

func (s *fakeStore) InsertStageRun(ctx context.Context, stage model.StageRun) error {
	return nil
}

func (s *fakeStore) UpdateStageRun(ctx context.Context, stage model.StageRun) error {
	return nil
}

func (s *fakeStore) InsertArtifact(ctx context.Context, projectId *string, run model.PipelineRun, stageName string, artifact model.ArtifactConfig, path string) error {
	return nil
}
