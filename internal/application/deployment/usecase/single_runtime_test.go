package deploymentsvc

import (
	"context"
	"strings"
	"testing"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestEnsureSingleRuntimeAppliesToGatewayServices(t *testing.T) {
	service := Service{store: &stores{service: singleRuntimeServiceStore{services: []model.Service{{
		Id: "service-default", ApplicationId: "gateway-1", InstanceKey: "default", Status: status.ServiceStatusRunning,
	}}}}}
	err := service.ensureSingleRuntime(context.Background(), model.Application{Id: "gateway-1", Kind: status.ApplicationKindGateway}, "staging")
	if err == nil || !strings.Contains(err.Error(), "single runtime only") {
		t.Fatalf("gateway concurrent runtime error = %v", err)
	}
}

type singleRuntimeServiceStore struct {
	repository.ServiceStore
	services []model.Service
}

func (s singleRuntimeServiceStore) ListServicesByApplication(context.Context, string) ([]model.Service, error) {
	return append([]model.Service(nil), s.services...), nil
}
