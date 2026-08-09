package repositorysvc

import (
	"context"
	"net/http"
	"strings"
	"testing"

	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
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
	if apperror.StatusCode(err) != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", apperror.StatusCode(err), http.StatusBadRequest)
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
