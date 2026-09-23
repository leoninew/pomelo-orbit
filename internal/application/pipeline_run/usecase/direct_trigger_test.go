package pipelinerunsvc

import (
	"context"
	"errors"
	"strings"
	"testing"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	pipelinerunport "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestTriggerPipelineRejectsMissingVariableBeforeCreatingSnapshot(t *testing.T) {
	projectId, repositoryId := "project-1", "repository-1"
	pipeline := model.Pipeline{
		Id:                   "pipeline-1",
		ProjectId:            &projectId,
		RepositoryId:         &repositoryId,
		Kind:                 model.PipelineKindApplication,
		VariableDeclarations: `[{"name":"IMAGE_TAG"}]`,
	}
	pipelineStore := &directTriggerPipelineStore{pipeline: pipeline}
	service := Service{store: stores{
		project:     directTriggerProjectStore{},
		repository:  directTriggerRepositoryStore{repository: model.Repository{Id: repositoryId, ProjectId: &projectId, DefaultBranch: "main"}},
		pipeline:    pipelineStore,
		pipelineRun: &directTriggerPipelineRunStore{},
	}}

	_, err := service.TriggerPipeline(context.Background(), "user-1", projectId, pipeline.Id, "release")
	if err == nil || !strings.Contains(err.Error(), "Missing variable value: IMAGE_TAG") {
		t.Fatalf("TriggerPipeline error = %v, want missing-variable validation", err)
	}
	if pipelineStore.snapshotRequested {
		t.Fatal("missing variables must be rejected before a snapshot is created")
	}
}

func TestTriggerPipelineRejectsUnconfiguredWorkspaceBeforeCreatingSnapshot(t *testing.T) {
	projectId, repositoryId := "project-1", "repository-1"
	pipeline := model.Pipeline{
		Id:                   "pipeline-1",
		ProjectId:            &projectId,
		RepositoryId:         &repositoryId,
		Kind:                 model.PipelineKindApplication,
		VariableDeclarations: `[]`,
	}
	pipelineStore := &directTriggerPipelineStore{pipeline: pipeline}
	runStore := &directTriggerPipelineRunStore{}
	service := Service{
		store: stores{
			project:     directTriggerProjectStore{},
			repository:  directTriggerRepositoryStore{repository: model.Repository{Id: repositoryId, ProjectId: &projectId, DefaultBranch: "main"}},
			pipeline:    pipelineStore,
			pipelineRun: runStore,
		},
		targetResolver: directTriggerTargetResolver{environment: model.Environment{Id: "env-1", ProjectId: projectId, TargetType: model.EnvironmentTargetTypeLocal, WorkspaceRoot: "/srv/orbit", TargetRevision: 1}},
		workspace:      directTriggerFailingWorkspace{err: errors.New("project environment is not configured")},
	}

	_, err := service.TriggerPipeline(context.Background(), "user-1", projectId, pipeline.Id, "release")
	if err == nil || !strings.Contains(err.Error(), "project environment is not configured") {
		t.Fatalf("TriggerPipeline error = %v, want unconfigured workspace validation", err)
	}
	if !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("TriggerPipeline error kind = %v, want %v", apperror.Kind(err.Error()), apperror.KindValidation)
	}
	if pipelineStore.snapshotRequested {
		t.Fatal("unconfigured workspace must be rejected before a snapshot is created")
	}
	if runStore.createdRun != nil {
		t.Fatal("unconfigured workspace must not insert a pipeline_run record")
	}
}

func TestRetryPipelineRunRejectsUnconfiguredWorkspaceBeforeCreatingSnapshot(t *testing.T) {
	projectId, repositoryId := "project-1", "repository-1"
	pipeline := model.Pipeline{
		Id:                   "pipeline-1",
		ProjectId:            &projectId,
		RepositoryId:         &repositoryId,
		Kind:                 model.PipelineKindApplication,
		VariableDeclarations: `[]`,
	}
	pipelineStore := &directTriggerPipelineStore{pipeline: pipeline}
	runStore := &directTriggerPipelineRunStore{
		run: model.PipelineRun{
			Id:                "run-1",
			ProjectId:         &projectId,
			RepositoryId:      repositoryId,
			PipelineId:        pipeline.Id,
			Status:            status.WorkStatusFaulted,
			VariablesSnapshot: `[]`,
		},
	}
	service := Service{
		store: stores{
			project:     directTriggerProjectStore{},
			repository:  directTriggerRepositoryStore{repository: model.Repository{Id: repositoryId, ProjectId: &projectId, DefaultBranch: "main"}},
			pipeline:    pipelineStore,
			pipelineRun: runStore,
		},
		targetResolver: directTriggerTargetResolver{environment: model.Environment{Id: "env-1", ProjectId: projectId, TargetType: model.EnvironmentTargetTypeLocal, WorkspaceRoot: "/srv/orbit", TargetRevision: 1}},
		workspace:      directTriggerFailingWorkspace{err: errors.New("project environment is not configured")},
	}

	_, err := service.RetryPipelineRun(context.Background(), "user-1", projectId, "run-1")
	if err == nil || !strings.Contains(err.Error(), "project environment is not configured") {
		t.Fatalf("RetryPipelineRun error = %v, want unconfigured workspace validation", err)
	}
	if !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("RetryPipelineRun error kind = %v, want %v", apperror.Kind(err.Error()), apperror.KindValidation)
	}
	if pipelineStore.snapshotRequested {
		t.Fatal("unconfigured workspace must be rejected before a snapshot is created on retry")
	}
	if runStore.createdRun != nil {
		t.Fatal("unconfigured workspace must not insert a pipeline_run record on retry")
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

func (s directTriggerRepositoryStore) Repository(context.Context, string, string) (model.Repository, error) {
	return s.repository, nil
}

type directTriggerPipelineStore struct {
	repository.PipelineStore
	pipeline          model.Pipeline
	snapshotRequested bool
}

func (s *directTriggerPipelineStore) Pipeline(context.Context, string, string) (model.Pipeline, error) {
	return s.pipeline, nil
}

func (s *directTriggerPipelineStore) ApplicationPipelineStages(context.Context, string, string) ([]model.PipelineStage, error) {
	return []model.PipelineStage{{Id: "stage-1", Name: "build", Script: "echo build"}}, nil
}

func (s *directTriggerPipelineStore) LatestPipelineSnapshot(context.Context, string, string) (model.PipelineSnapshot, error) {
	s.snapshotRequested = true
	return model.PipelineSnapshot{}, repository.ErrNotFound
}

type directTriggerPipelineRunStore struct {
	repository.PipelineRunStore
	run        model.PipelineRun
	createdRun *model.PipelineRun
}

func (s *directTriggerPipelineRunStore) PipelineRun(context.Context, string, string) (model.PipelineRun, error) {
	if s.run.Id != "" {
		return s.run, nil
	}
	return model.PipelineRun{}, repository.ErrNotFound
}

func (s *directTriggerPipelineRunStore) CreatePipelineRun(_ context.Context, run model.PipelineRun, _ *model.PipelineRunVersionBinding, _ []model.PipelineStageRun) error {
	s.createdRun = &run
	return nil
}

func (directTriggerPipelineRunStore) RepositoryHasActivePipelineRun(context.Context, string, string) (bool, error) {
	return false, nil
}

type directTriggerFailingWorkspace struct {
	err error
}

func (w directTriggerFailingWorkspace) WorkspaceForProject(context.Context, string) (pipelinerunport.Workspace, error) {
	return nil, w.err
}

func (directTriggerFailingWorkspace) CreateRunDirectories(string, string) error   { return nil }
func (directTriggerFailingWorkspace) ArtifactsPath(string) string                 { return "" }
func (directTriggerFailingWorkspace) ArtifactExists(string, string) (bool, error) { return false, nil }
func (directTriggerFailingWorkspace) StageLogPath(string, string) string          { return "" }
func (directTriggerFailingWorkspace) RemoveRunFiles(string) error                 { return nil }
func (directTriggerFailingWorkspace) DockerStageMounts(context.Context, string, string) ([]pipelinerunport.VolumeMount, error) {
	return nil, nil
}

type directTriggerTargetResolver struct {
	environment model.Environment
	err         error
}

func (resolver directTriggerTargetResolver) ResolveProjectTarget(context.Context, string) (environmentport.Target, error) {
	return environmentport.Target{Environment: resolver.environment}, resolver.err
}

func (w directTriggerFailingWorkspace) WorkspaceForTarget(context.Context, model.Environment) (pipelinerunport.Workspace, error) {
	return nil, w.err
}
