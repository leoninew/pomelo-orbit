package repositorysvc

import (
	"context"
	"strings"
	"testing"

	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestDeleteRepositoryRejectsBoundPipelinesWithValidation(t *testing.T) {
	projectId := "project-1"
	repositoryStore := &repositoryDeletionStore{
		item:  model.Repository{Id: "repository-1", ProjectId: &projectId},
		bound: true,
	}
	service := Service{store: stores{
		project:    repositoryDeletionProjectStore{},
		repository: repositoryStore,
	}}

	err := service.DeleteRepository(context.Background(), "user-1", projectId, "repository-1")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("error = %v, want validation", err)
	}
	if !strings.Contains(err.Error(), "bound to pipelines") {
		t.Fatalf("error = %q, want pipeline binding guidance", err)
	}
	if repositoryStore.deleted {
		t.Fatal("repository must remain while pipelines are bound")
	}
	if !repositoryStore.boundChecked {
		t.Fatal("repository pipeline bindings must be checked before deletion")
	}
}

func TestDeleteRepositoryAllowsDeletionWhenNoPipelineIsBound(t *testing.T) {
	projectId := "project-1"
	repositoryStore := &repositoryDeletionStore{
		item: model.Repository{Id: "repository-1", ProjectId: &projectId},
	}
	service := Service{store: stores{
		project:    repositoryDeletionProjectStore{},
		repository: repositoryStore,
	}}

	if err := service.DeleteRepository(context.Background(), "user-1", projectId, "repository-1"); err != nil {
		t.Fatalf("delete repository: %v", err)
	}
	if !repositoryStore.boundChecked {
		t.Fatal("repository pipeline bindings must be checked before deletion")
	}
	if !repositoryStore.deleted {
		t.Fatal("repository should be deleted when no pipeline is bound")
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
	item         model.Repository
	bound        bool
	boundChecked bool
	deleted      bool
}

func (s *repositoryDeletionStore) Repository(context.Context, string, string) (model.Repository, error) {
	return s.item, nil
}

func (s *repositoryDeletionStore) RepositoryHasBoundPipelines(context.Context, string, string) (bool, error) {
	s.boundChecked = true
	return s.bound, nil
}

func (s *repositoryDeletionStore) DeleteRepository(context.Context, string, string) error {
	s.deleted = true
	return nil
}
