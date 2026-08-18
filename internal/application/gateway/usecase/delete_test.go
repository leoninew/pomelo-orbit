package gatewaysvc

import (
	"context"
	"net/http"
	"testing"

	gatewayport "github.com/leoninew/pomelo-orbit/internal/application/gateway/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type gatewayDeleteApplicationStore struct {
	gatewayport.ApplicationStore
	application model.Application
	deleteId    string
	deleteErr   error
}

func (s *gatewayDeleteApplicationStore) Application(_ context.Context, _ string) (model.Application, error) {
	return s.application, nil
}

func (s *gatewayDeleteApplicationStore) ListApplications(_ context.Context, _ *string, _ int, _ int, _ string, _ string) (repository.Page[model.Application], error) {
	return repository.Page[model.Application]{}, nil
}

func (s *gatewayDeleteApplicationStore) DeleteGatewayApplication(_ context.Context, id string) error {
	s.deleteId = id
	return s.deleteErr
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

func (s gatewayDeleteServiceStore) ListServicesByApplication(_ context.Context, _ string) ([]model.Service, error) {
	return s.services, nil
}

func TestDeleteGatewayDeletesStoppedServiceResources(t *testing.T) {
	application := &gatewayDeleteApplicationStore{application: model.Application{Id: "gateway-1", Kind: status.ApplicationKindGateway}}
	service := New(
		nil,
		application,
		gatewayDeleteConfigStore{config: model.GatewayConfig{ApplicationId: "gateway-1"}},
		gatewayDeleteServiceStore{services: []model.Service{{Id: "service-1", InstanceKey: "default", Status: status.ServiceStatusStopped}}},
		nil,
		testGatewayConfig(),
		nil,
		nil,
	)

	if err := service.DeleteGateway(context.Background(), "user-1", "gateway-1"); err != nil {
		t.Fatal(err)
	}
	if application.deleteId != "gateway-1" {
		t.Fatalf("deleted application = %q, want gateway-1", application.deleteId)
	}
}

func TestDeleteGatewayRejectsNonStoppedService(t *testing.T) {
	for _, serviceStatus := range []string{status.ServiceStatusRunning, status.ServiceStatusFaulted} {
		t.Run(serviceStatus, func(t *testing.T) {
			application := &gatewayDeleteApplicationStore{application: model.Application{Id: "gateway-1", Kind: status.ApplicationKindGateway}}
			service := New(
				nil,
				application,
				gatewayDeleteConfigStore{config: model.GatewayConfig{ApplicationId: "gateway-1"}},
				gatewayDeleteServiceStore{services: []model.Service{{Id: "service-1", InstanceKey: "default", Code: "gateway-default", Status: serviceStatus}}},
				nil,
				testGatewayConfig(),
				nil,
				nil,
			)

			err := service.DeleteGateway(context.Background(), "user-1", "gateway-1")
			classification := apperror.Classify(err)
			if classification.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; error = %v", classification.StatusCode, http.StatusBadRequest, err)
			}
			if classification.Message != "网关存在未停止的服务 gateway-default, 请先停止后再删除" {
				t.Fatalf("message = %q", classification.Message)
			}
			if application.deleteId != "" {
				t.Fatalf("unexpected deletion of %q", application.deleteId)
			}
		})
	}
}
