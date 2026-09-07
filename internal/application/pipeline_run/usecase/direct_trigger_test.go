package pipelinerunsvc

import (
	"context"
	"strings"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestTriggerPipelineRejectsMissingVariableBeforeCreatingSnapshot(t *testing.T) {
	projectID, repositoryID := "project-1", "repository-1"
	pipeline := model.Pipeline{
		Id:                   "pipeline-1",
		ProjectId:            &projectID,
		RepositoryId:         &repositoryID,
		Kind:                 model.PipelineKindApplication,
		VariableDeclarations: `[{"name":"IMAGE_TAG"}]`,
	}
	pipelineStore := &directTriggerPipelineStore{pipeline: pipeline}
	service := Service{store: stores{
		project:     directTriggerProjectStore{},
		repository:  directTriggerRepositoryStore{repository: model.Repository{Id: repositoryID, ProjectId: &projectID, DefaultBranch: "main"}},
		pipeline:    pipelineStore,
		pipelineRun: directTriggerPipelineRunStore{},
	}}

	_, err := service.TriggerPipeline(context.Background(), "user-1", pipeline.Id, "release")
	if err == nil || !strings.Contains(err.Error(), "Missing variable value: IMAGE_TAG") {
		t.Fatalf("TriggerPipeline error = %v, want missing-variable validation", err)
	}
	if pipelineStore.snapshotRequested {
		t.Fatal("missing variables must be rejected before a snapshot is created")
	}
}

type directTriggerProjectStore struct {
	repository.ProjectReader
}

func (directTriggerProjectStore) Project(context.Context, string) (model.Project, error) {
	return model.Project{Id: "project-1"}, nil
}

func (directTriggerProjectStore) IsProjectMember(context.Context, string, string) (bool, error) {
	return true, nil
}

type directTriggerRepositoryStore struct {
	repository.RepositoryStore
	repository model.Repository
}

func (s directTriggerRepositoryStore) Repository(context.Context, string) (model.Repository, error) {
	return s.repository, nil
}

type directTriggerPipelineStore struct {
	repository.PipelineStore
	pipeline          model.Pipeline
	snapshotRequested bool
}

func (s *directTriggerPipelineStore) Pipeline(context.Context, string) (model.Pipeline, error) {
	return s.pipeline, nil
}

func (s *directTriggerPipelineStore) ApplicationPipelineStages(context.Context, string) ([]model.PipelineStage, error) {
	return []model.PipelineStage{{Id: "stage-1", Name: "build", Script: "echo build"}}, nil
}

func (s *directTriggerPipelineStore) LatestPipelineSnapshot(context.Context, string) (model.PipelineSnapshot, error) {
	s.snapshotRequested = true
	return model.PipelineSnapshot{}, repository.ErrNotFound
}

type directTriggerPipelineRunStore struct {
	repository.PipelineRunStore
}

func (directTriggerPipelineRunStore) RepositoryHasActivePipelineRun(context.Context, string) (bool, error) {
	return false, nil
}
