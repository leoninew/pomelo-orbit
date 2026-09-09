package repositorysvc

import (
	"context"
	"strings"
	"testing"

	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestDeleteRepositoryRejectsRunningPipelinesWithValidation(t *testing.T) {
	projectID := "project-1"
	repositoryStore := &repositoryDeletionStore{
		item:    model.Repository{Id: "repository-1", ProjectId: &projectID},
		running: true,
	}
	service := Service{store: stores{
		project:    repositoryDeletionProjectStore{},
		repository: repositoryStore,
	}}

	err := service.DeleteRepository(context.Background(), "user-1", "repository-1")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("error = %v, want validation", err)
	}
	if !strings.Contains(err.Error(), "Cancel or wait") {
		t.Fatalf("error = %q, want actionable guidance", err)
	}
	if repositoryStore.deleted {
		t.Fatal("repository must remain while pipelines are running")
	}
}

type repositoryDeletionProjectStore struct {
	repository.ProjectReader
}

func (repositoryDeletionProjectStore) Project(_ context.Context, id string) (model.Project, error) {
	return model.Project{Id: id}, nil
}

func (repositoryDeletionProjectStore) IsProjectMember(context.Context, string, string) (bool, error) {
	return true, nil
}

type repositoryDeletionStore struct {
	repository.RepositoryStore
	item    model.Repository
	running bool
	deleted bool
}

func (s *repositoryDeletionStore) Repository(context.Context, string) (model.Repository, error) {
	return s.item, nil
}

func (s *repositoryDeletionStore) RepositoryHasRunningPipelines(context.Context, string) (bool, error) {
	return s.running, nil
}

func (s *repositoryDeletionStore) DeleteRepository(context.Context, string) error {
	s.deleted = true
	return nil
}
