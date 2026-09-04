package deploymentsvc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestDeleteDeploymentRemovesTerminalRecordWhenLogFileIsMissing(t *testing.T) {
	deployment := deploymentForDeletion(status.WorkStatusRanToCompletion)
	store := &deploymentDeletionStore{deployment: deployment}
	workspace := testWorkspace(t.TempDir())
	workspace.SetMissingDeploymentLog()
	service := newDeploymentDeletionService(store, &deploymentDeletionServiceStore{
		service: model.Service{Id: *deployment.ServiceId, Code: "demo-default"},
	}, workspace)

	if err := service.DeleteDeployment(context.Background(), "user-1", deployment.Id); err != nil {
		t.Fatalf("DeleteDeployment returned error: %v", err)
	}
	if !store.deleted {
		t.Fatal("deployment record was not deleted")
	}
	if got, want := workspace.removedLog, (workspaceRemovedLog{serviceCode: "demo-default", deploymentID: deployment.Id}); got != want {
		t.Fatalf("removed log = %#v, want %#v", got, want)
	}
}

func TestDeleteDeploymentRejectsActiveRecord(t *testing.T) {
	deployment := deploymentForDeletion(status.WorkStatusRunning)
	store := &deploymentDeletionStore{deployment: deployment}
	workspace := testWorkspace(t.TempDir())
	service := newDeploymentDeletionService(store, &deploymentDeletionServiceStore{
		service: model.Service{Id: *deployment.ServiceId, Code: "demo-default"},
	}, workspace)

	err := service.DeleteDeployment(context.Background(), "user-1", deployment.Id)
	if err == nil || !strings.Contains(err.Error(), "Cannot delete deployment") {
		t.Fatalf("DeleteDeployment error = %v, want active deployment validation error", err)
	}
	if store.deleted {
		t.Fatal("active deployment record must not be deleted")
	}
	if workspace.removedLog != (workspaceRemovedLog{}) {
		t.Fatal("active deployment log must not be removed")
	}
}

func TestDeleteDeploymentKeepsRecordWhenLogRemovalFails(t *testing.T) {
	deployment := deploymentForDeletion(status.WorkStatusFaulted)
	store := &deploymentDeletionStore{deployment: deployment}
	workspace := testWorkspace(t.TempDir())
	workspace.removeLogErr = errors.New("permission denied")
	service := newDeploymentDeletionService(store, &deploymentDeletionServiceStore{
		service: model.Service{Id: *deployment.ServiceId, Code: "demo-default"},
	}, workspace)

	err := service.DeleteDeployment(context.Background(), "user-1", deployment.Id)
	if err == nil || !strings.Contains(err.Error(), "Failed to delete deployment log") {
		t.Fatalf("DeleteDeployment error = %v, want log removal error", err)
	}
	if store.deleted {
		t.Fatal("deployment record must remain when log removal fails")
	}
}

func TestDeleteDeploymentRemovesRecordWhenAssociatedServiceIsMissing(t *testing.T) {
	deployment := deploymentForDeletion(status.WorkStatusCanceled)
	store := &deploymentDeletionStore{deployment: deployment}
	service := newDeploymentDeletionService(store, &deploymentDeletionServiceStore{
		err: repository.ErrNotFound,
	}, testWorkspace(t.TempDir()))

	if err := service.DeleteDeployment(context.Background(), "user-1", deployment.Id); err != nil {
		t.Fatalf("DeleteDeployment returned error: %v", err)
	}
	if !store.deleted {
		t.Fatal("deployment record was not deleted")
	}
}

func deploymentForDeletion(deploymentStatus string) model.Deployment {
	projectID := "project-1"
	serviceID := "service-1"
	return model.Deployment{
		Id:        "deployment-1",
		ProjectId: &projectID,
		ServiceId: &serviceID,
		Status:    deploymentStatus,
	}
}

func newDeploymentDeletionService(store *deploymentDeletionStore, serviceStore *deploymentDeletionServiceStore, workspace *workspaceFake) Service {
	return Service{
		project:    deploymentDeletionProjectStore{},
		service:    serviceStore,
		deployment: store,
		logStore:   workspace,
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

type deploymentDeletionProjectStore struct {
	repository.ProjectReader
}

func (deploymentDeletionProjectStore) Project(context.Context, string) (model.Project, error) {
	return model.Project{Id: "project-1"}, nil
}

func (deploymentDeletionProjectStore) IsProjectMember(context.Context, string, string) (bool, error) {
	return true, nil
}

type deploymentDeletionServiceStore struct {
	repository.ServiceStore
	service model.Service
	err     error
}

func (s *deploymentDeletionServiceStore) Service(context.Context, string) (model.Service, error) {
	return s.service, s.err
}

type deploymentDeletionStore struct {
	repository.DeploymentStore
	deployment model.Deployment
	deleted    bool
}

func (s *deploymentDeletionStore) Deployment(context.Context, string) (model.Deployment, error) {
	return s.deployment, nil
}

func (s *deploymentDeletionStore) DeleteDeployment(context.Context, string) error {
	s.deleted = true
	return nil
}
