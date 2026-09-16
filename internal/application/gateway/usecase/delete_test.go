package gatewaysvc

import (
	"context"
	"testing"

	gatewayport "github.com/leoninew/pomelo-orbit/internal/application/gateway/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type gatewayDeleteProjectStore struct {
	gatewayport.ProjectReader
	project model.Project
}

func (s gatewayDeleteProjectStore) Project(context.Context, string) (model.Project, error) {
	return s.project, nil
}

func (gatewayDeleteProjectStore) IsProjectMember(context.Context, string, string) (bool, error) {
	return true, nil
}

type gatewayDeleteEnvironmentStore struct {
	gatewayport.EnvironmentStore
	environment model.Environment
}

func (s gatewayDeleteEnvironmentStore) EnvironmentByProject(context.Context, string) (model.Environment, error) {
	return s.environment, nil
}

type gatewayDeleteApplicationStore struct {
	gatewayport.ApplicationStore
	application model.Application
	deleteId    string
}

func (s *gatewayDeleteApplicationStore) Application(_ context.Context, _, _ string) (model.Application, error) {
	return s.application, nil
}

func (s *gatewayDeleteApplicationStore) ListApplications(_ context.Context, _ string, _ int, _ int, _ string, _ string) (repository.Page[model.Application], error) {
	return repository.Page[model.Application]{}, nil
}

func (s *gatewayDeleteApplicationStore) DeleteApplication(_ context.Context, id string) error {
	s.deleteId = id
	return nil
}

type gatewayDeleteConfigStore struct {
	gatewayport.ConfigStore
	config model.GatewayConfig
}

func (s gatewayDeleteConfigStore) GatewayConfig(_ context.Context, _ string) (model.GatewayConfig, error) {
	return s.config, nil
}

type gatewayDeleteServiceStore struct {
	repository.ServiceStore
	services []model.Service
}

func (s gatewayDeleteServiceStore) ListServicesByApplication(_ context.Context, _, _ string) ([]model.Service, error) {
	return s.services, nil
}

func TestDeleteGatewayRejectsBoundGateway(t *testing.T) {
	projectId := "project-1"
	gatewayId := "gateway-1"
	application := &gatewayDeleteApplicationStore{application: model.Application{Id: gatewayId, ProjectId: &projectId, Kind: status.ApplicationKindGateway}}
	service := New(
		gatewayDeleteProjectStore{project: model.Project{Id: projectId}},
		gatewayDeleteEnvironmentStore{environment: model.Environment{ProjectId: projectId, GatewayApplicationId: &gatewayId}},
		application,
		gatewayDeleteConfigStore{config: model.GatewayConfig{ApplicationId: gatewayId}},
		gatewayDeleteServiceStore{services: []model.Service{{Id: "service-1", InstanceKey: "default", Status: status.ServiceStatusStopped}}},
		nil,
		nil,
		nil,
		nil,
	)

	err := service.DeleteGateway(context.Background(), "user-1", projectId, gatewayId)
	if !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("error = %v, want validation", err)
	}
	if apperror.Classify(err).Message != "Gateway cannot be deleted after it is bound to a project environment" {
		t.Fatalf("message = %q", apperror.Classify(err).Message)
	}
	if application.deleteId != "" {
		t.Fatalf("unexpected deletion of %q", application.deleteId)
	}
}
