package servicesvc

import (
	"context"
	"testing"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func TestDeleteServiceDeletesStoppedService(t *testing.T) {
	service, store := newDeleteServiceTestService(status.ServiceStatusStopped)

	err := service.DeleteService(context.Background(), "user-1", "service-1")
	if err != nil {
		t.Fatalf("DeleteService returned error: %v", err)
	}
	if store.deletedID != "service-1" {
		t.Fatalf("deleted service id = %q, want service-1", store.deletedID)
	}
}

func TestDeleteServiceRejectsNonStoppedService(t *testing.T) {
	service, store := newDeleteServiceTestService(status.ServiceStatusRunning)

	err := service.DeleteService(context.Background(), "user-1", "service-1")
	if apperror.StatusCode(err) != 400 {
		t.Fatalf("DeleteService error status = %d, want 400; error=%v", apperror.StatusCode(err), err)
	}
	if store.deletedID != "" {
		t.Fatalf("service must not be deleted while running: %q", store.deletedID)
	}
}

func newDeleteServiceTestService(serviceStatus string) (Service, *deleteServiceStoreFake) {
	projectID := "project-1"
	store := &deleteServiceStoreFake{item: model.ServiceListItem{
		Id: "service-1", ApplicationId: "application-1", InstanceKey: "default", VersionId: "version-1", Status: serviceStatus,
	}}
	return New(
		deleteServiceProjectFake{},
		deleteServiceApplicationFake{application: model.Application{Id: "application-1", ProjectId: &projectID}},
		store,
	), store
}

type deleteServiceProjectFake struct{}

func (deleteServiceProjectFake) Project(_ context.Context, id string) (model.Project, error) {
	return model.Project{Id: id}, nil
}

func (deleteServiceProjectFake) IsProjectMember(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}

type deleteServiceApplicationFake struct {
	application model.Application
}

func (f deleteServiceApplicationFake) Application(_ context.Context, _ string) (model.Application, error) {
	return f.application, nil
}

func (deleteServiceApplicationFake) Version(_ context.Context, _ string) (model.Version, error) {
	return model.Version{}, repository.ErrNotFound
}

func (deleteServiceApplicationFake) VersionComponentsByVersion(_ context.Context, _ string) ([]model.VersionComponent, error) {
	return nil, nil
}

type deleteServiceStoreFake struct {
	item      model.ServiceListItem
	deletedID string
}

func (f *deleteServiceStoreFake) ListServicesByApplication(_ context.Context, _ string) ([]model.Service, error) {
	return []model.Service{f.item.Service()}, nil
}

func (f *deleteServiceStoreFake) ListServicesByProject(_ context.Context, _, _, _, _ string, page, perPage int) (repository.Page[model.ServiceListItem], error) {
	return repository.Page[model.ServiceListItem]{Items: []model.ServiceListItem{f.item}, Total: 1, Page: page, PerPage: perPage}, nil
}

func (f *deleteServiceStoreFake) ServiceListItem(_ context.Context, _ string) (model.ServiceListItem, error) {
	return f.item, nil
}

func (f *deleteServiceStoreFake) ServiceByKey(_ context.Context, _, _ string) (model.Service, error) {
	return f.item.Service(), nil
}

func (f *deleteServiceStoreFake) Service(_ context.Context, _ string) (model.Service, error) {
	return f.item.Service(), nil
}

func (f *deleteServiceStoreFake) DeleteService(_ context.Context, id string) error {
	f.deletedID = id
	return nil
}

func (f *deleteServiceStoreFake) UpsertService(_ context.Context, _ model.Service) error { return nil }

func (f *deleteServiceStoreFake) UpdateServiceRuntimeConfig(_ context.Context, _ string, _ map[string]string) error {
	return nil
}
