package servicesvc

import (
	"context"
	"strings"
	"testing"

	servicedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type serviceCreateStoreFake struct {
	repository.ServiceStore
	serviceByKeyErr  error
	serviceByCodeErr error
	created          model.Service
}

func (f *serviceCreateStoreFake) ServiceByKey(context.Context, string, string) (model.Service, error) {
	return model.Service{}, f.serviceByKeyErr
}

func (f *serviceCreateStoreFake) ServiceByCode(context.Context, string) (model.Service, error) {
	return model.Service{}, f.serviceByCodeErr
}

func (f *serviceCreateStoreFake) CreateServiceWithComponents(_ context.Context, service model.Service, _ []model.ServiceComponent) error {
	f.created = service
	return nil
}

func (f *serviceCreateStoreFake) ServiceListItem(context.Context, string) (model.ServiceListItem, error) {
	return model.ServiceListItem{
		Id: f.created.Id, ApplicationId: f.created.ApplicationId, InstanceKey: f.created.InstanceKey, Code: f.created.Code,
		VersionId: f.created.VersionId, Status: f.created.Status, ApplicationName: "RAGFlow", ApplicationCode: "ragflow",
	}, nil
}

func (f *serviceCreateStoreFake) ServiceEnvByService(context.Context, string) ([]model.ServiceEnv, error) {
	return nil, nil
}

func (f *serviceCreateStoreFake) ServiceComponentsByService(context.Context, string) ([]model.ServiceComponent, error) {
	return nil, nil
}

func TestCreateServiceUsesSubmittedCode(t *testing.T) {
	store := &serviceCreateStoreFake{serviceByKeyErr: repository.ErrNotFound, serviceByCodeErr: repository.ErrNotFound}
	service := Service{
		application: serviceApplicationFake{
			app:     model.Application{Id: "app-1", Code: "ragflow"},
			version: model.Version{Id: "version-1", ApplicationId: "app-1"},
			declarations: []model.VersionComponent{{
				Id: "component-1", Name: "ragflow", Image: "ragflow:latest",
			}},
		},
		service: store,
	}

	view, err := service.CreateService(context.Background(), "user-1", servicedto.ServiceCreateInput{ApplicationId: "app-1", VersionId: "version-1", InstanceKey: "default", Code: "ragflow-preview"})
	if err != nil {
		t.Fatal(err)
	}
	if store.created.Code != "ragflow-preview" || view.Service.Code != "ragflow-preview" {
		t.Fatalf("service code was not preserved: created=%q view=%q", store.created.Code, view.Service.Code)
	}
}

func TestCreateServiceReportsExistingCodeConflict(t *testing.T) {
	store := &serviceCreateStoreFake{serviceByKeyErr: repository.ErrNotFound}
	service := Service{
		application: serviceApplicationFake{
			app: model.Application{Id: "app-1", Code: "ragflow"}, version: model.Version{Id: "version-1", ApplicationId: "app-1"},
		},
		service: store,
	}

	_, err := service.CreateService(context.Background(), "user-1", servicedto.ServiceCreateInput{ApplicationId: "app-1", VersionId: "version-1", InstanceKey: "default", Code: "ragflow-default"})
	if err == nil || !strings.Contains(err.Error(), "Service code already exists") {
		t.Fatalf("create error = %v", err)
	}
}

func TestNormalizeServiceCode(t *testing.T) {
	code, err := normalizeServiceCode(" ragflow-default ")
	if err != nil || code != "ragflow-default" {
		t.Fatalf("code = %q, err = %v", code, err)
	}
	for _, input := range []string{"ragflow_default", "-ragflow", strings.Repeat("a", 64)} {
		if _, err := normalizeServiceCode(input); err == nil {
			t.Fatalf("service code %q must be rejected", input)
		}
	}
}
