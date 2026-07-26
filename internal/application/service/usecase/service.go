package servicesvc

import (
	"context"
	"errors"
	"strings"

	servicedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/dto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type Service struct {
	project     repository.ProjectReader
	application repository.ApplicationReader
	service     repository.ServiceReader
}

func New(project repository.ProjectReader, application repository.ApplicationReader, service repository.ServiceReader) Service {
	return Service{project: project, application: application, service: service}
}

// ListServices returns runtime bindings for a project.
func (s Service) ListServices(ctx context.Context, userId string, input servicedto.ServiceListInput) (repository.Page[servicedto.ServiceView], error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return repository.Page[servicedto.ServiceView]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[servicedto.ServiceView]{}, err
	}
	page, err := s.service.ListServicesByProject(ctx, projectId, input.ApplicationId, input.Status, input.Search, input.Page, input.PerPage)
	if err != nil {
		return repository.Page[servicedto.ServiceView]{}, apperror.Wrap(apperror.KindInternal, "Failed to list services", err)
	}
	items := make([]servicedto.ServiceView, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, serviceViewFromListItem(item))
	}
	return repository.Page[servicedto.ServiceView]{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage}, nil
}

// GetService returns one runtime binding with display labels when the user can access its application.
func (s Service) GetService(ctx context.Context, userId string, serviceId string) (servicedto.ServiceView, error) {
	serviceId = strings.TrimSpace(serviceId)
	if serviceId == "" {
		return servicedto.ServiceView{}, apperror.New(apperror.KindValidation, "service_id is required")
	}
	item, err := s.service.ServiceListItem(ctx, serviceId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return servicedto.ServiceView{}, apperror.New(apperror.KindNotFound, "Service "+serviceId+" not found")
		}
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	if _, err := s.loadApplicationForUser(ctx, userId, item.ApplicationId); err != nil {
		return servicedto.ServiceView{}, err
	}
	return serviceViewFromListItem(item), nil
}

// ListServicesByApplication returns runtime bindings after authorizing access to the application.
func (s Service) ListServicesByApplication(ctx context.Context, userId string, applicationId string) ([]model.Service, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return nil, err
	}
	services, err := s.service.ListServicesByApplication(ctx, app.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list services", err)
	}
	return services, nil
}

// PrimaryServiceByApplication returns the first runtime binding, if the application is running.
func (s Service) PrimaryServiceByApplication(ctx context.Context, userId string, applicationId string) (*model.Service, error) {
	services, err := s.ListServicesByApplication(ctx, userId, applicationId)
	if err != nil {
		return nil, err
	}
	if len(services) == 0 {
		return nil, nil
	}
	return &services[0], nil
}

// ResolveServiceTarget authorizes the application and resolves an unambiguous runtime binding.
func (s Service) ResolveServiceTarget(ctx context.Context, userId string, applicationId string, input servicedto.ServiceTargetInput) (model.Service, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return model.Service{}, err
	}
	return s.resolveServiceTarget(ctx, app.Id, input)
}

func (s Service) resolveServiceTarget(ctx context.Context, applicationId string, input servicedto.ServiceTargetInput) (model.Service, error) {
	if serviceId := strings.TrimSpace(input.ServiceId); serviceId != "" {
		svc, err := s.service.Service(ctx, serviceId)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return model.Service{}, apperror.New(apperror.KindNotFound, "Service not found")
			}
			return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
		}
		if svc.ApplicationId != applicationId {
			return model.Service{}, apperror.New(apperror.KindNotFound, "Service not found")
		}
		return svc, nil
	}

	if input.InstanceKey == "" {
		return model.Service{}, apperror.New(apperror.KindValidation, "instance_key is required")
	}

	svc, err := s.service.ServiceByKey(ctx, applicationId, input.InstanceKey)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Service{}, apperror.New(apperror.KindValidation, "应用未在运行中")
		}
		return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	return svc, nil
}

func serviceViewFromListItem(item model.ServiceListItem) servicedto.ServiceView {
	return servicedto.ServiceView{
		Service:                    item.Service(),
		ApplicationName:            item.ApplicationName,
		ApplicationCode:            item.ApplicationCode,
		ApplicationKind:            item.ApplicationKind,
		VersionLabel:               item.VersionLabel,
		LastSuccessfulVersionLabel: item.LastSuccessfulVersionLabel,
	}
}

func (s Service) loadApplicationForUser(ctx context.Context, userId string, applicationId string) (model.Application, error) {
	applicationId = strings.TrimSpace(applicationId)
	app, err := s.application.Application(ctx, applicationId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Application{}, apperror.New(apperror.KindNotFound, "Application "+applicationId+" not found")
		}
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	if app.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *app.ProjectId, userId); err != nil {
			return model.Application{}, err
		}
	}
	return app, nil
}

func (s Service) ensureProjectMembership(ctx context.Context, projectId string, userId string) error {
	if _, err := s.project.Project(ctx, projectId); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.project.IsProjectMember(ctx, projectId, userId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return nil
}
