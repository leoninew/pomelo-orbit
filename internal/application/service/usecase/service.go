package servicesvc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	servicedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	runtimeconfig "gitee.com/leoninew/PomeloOrbit-go/internal/common/runtimeconfig"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type applicationReader interface {
	Application(ctx context.Context, id string) (model.Application, error)
	Version(ctx context.Context, id string) (model.Version, error)
	VersionComponentsByVersion(ctx context.Context, versionId string) ([]model.VersionComponent, error)
}

type serviceStore interface {
	repository.ServiceReader
	CreateServiceWithExposes(ctx context.Context, service model.Service, exposes []model.ServiceExpose) error
	UpdateServiceConfiguration(ctx context.Context, service model.Service, exposes []model.ServiceExpose) error
	DeleteService(ctx context.Context, id string) error
}

type Service struct {
	project     repository.ProjectReader
	application applicationReader
	service     serviceStore
}

func New(project repository.ProjectReader, application applicationReader, service serviceStore) Service {
	return Service{
		project: project, application: application, service: service,
	}
}

// DeleteService removes a stopped runtime binding while preserving its deployment history.
func (s Service) DeleteService(ctx context.Context, userId string, serviceId string) error {
	serviceId = strings.TrimSpace(serviceId)
	if serviceId == "" {
		return apperror.New(apperror.KindValidation, "service_id is required")
	}
	item, err := s.service.ServiceListItem(ctx, serviceId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Service "+serviceId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	if _, err := s.loadApplicationForUser(ctx, userId, item.ApplicationId); err != nil {
		return err
	}
	if item.Status != status.ServiceStatusStopped {
		return apperror.New(apperror.KindValidation, "Only stopped services can be deleted")
	}
	if err := s.service.DeleteService(ctx, item.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete service", err)
	}
	return nil
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
		view, err := s.serviceView(ctx, item)
		if err != nil {
			return repository.Page[servicedto.ServiceView]{}, err
		}
		items = append(items, view)
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
	return s.serviceView(ctx, item)
}

func (s Service) CreateService(ctx context.Context, userId string, input servicedto.ServiceCreateInput) (servicedto.ServiceView, error) {
	applicationId := strings.TrimSpace(input.ApplicationId)
	versionId := strings.TrimSpace(input.VersionId)
	instanceKey := strings.TrimSpace(input.InstanceKey)
	if applicationId == "" || versionId == "" || instanceKey == "" {
		return servicedto.ServiceView{}, apperror.New(apperror.KindValidation, "application_id, version_id and instance_key are required")
	}
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return servicedto.ServiceView{}, err
	}
	version, components, err := s.versionComponents(ctx, versionId, app.Id)
	if err != nil {
		return servicedto.ServiceView{}, err
	}
	runtimeConfig, err := normalizeRuntimeConfig(input.RuntimeConfig)
	if err != nil {
		return servicedto.ServiceView{}, err
	}
	if err := validateRuntimeConfig(runtimeConfig, components); err != nil {
		return servicedto.ServiceView{}, err
	}
	exposes, err := serviceExposesFromInputs(input.Exposes)
	if err != nil {
		return servicedto.ServiceView{}, err
	}
	if err := validateServiceExposes(exposes, components); err != nil {
		return servicedto.ServiceView{}, err
	}
	if err := s.validateExposeConflicts(ctx, "", exposes); err != nil {
		return servicedto.ServiceView{}, err
	}
	if _, err := s.service.ServiceByKey(ctx, app.Id, instanceKey); err == nil {
		return servicedto.ServiceView{}, apperror.New(apperror.KindConflict, "Service instance already exists")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	svc := model.Service{Id: idutil.NewId(), ApplicationId: app.Id, InstanceKey: instanceKey, VersionId: version.Id, RuntimeConfig: runtimeConfig, Status: status.ServiceStatusStopped}
	for i := range exposes {
		exposes[i].Id = idutil.NewId()
		exposes[i].ServiceId = svc.Id
	}
	if err := s.service.CreateServiceWithExposes(ctx, svc, exposes); err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to create service", err)
	}
	created, err := s.service.ServiceListItem(ctx, svc.Id)
	if err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	return s.serviceView(ctx, created)
}

func (s Service) UpdateServiceConfiguration(ctx context.Context, userId string, serviceId string, input servicedto.ServiceConfigInput) (servicedto.ServiceView, error) {
	svc, err := s.serviceForUser(ctx, userId, serviceId)
	if err != nil {
		return servicedto.ServiceView{}, err
	}
	_, components, err := s.versionComponents(ctx, svc.VersionId, svc.ApplicationId)
	if err != nil {
		return servicedto.ServiceView{}, err
	}
	runtimeConfig, err := normalizeRuntimeConfig(input.RuntimeConfig)
	if err != nil {
		return servicedto.ServiceView{}, err
	}
	if err := validateRuntimeConfig(runtimeConfig, components); err != nil {
		return servicedto.ServiceView{}, err
	}
	exposes, err := serviceExposesFromInputs(input.Exposes)
	if err != nil {
		return servicedto.ServiceView{}, err
	}
	if err := validateServiceExposes(exposes, components); err != nil {
		return servicedto.ServiceView{}, err
	}
	if err := s.validateExposeConflicts(ctx, svc.Id, exposes); err != nil {
		return servicedto.ServiceView{}, err
	}
	for i := range exposes {
		exposes[i].Id = idutil.NewId()
		exposes[i].ServiceId = svc.Id
	}
	svc.RuntimeConfig = runtimeConfig
	if err := s.service.UpdateServiceConfiguration(ctx, svc, exposes); err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to update service configuration", err)
	}
	updated, err := s.service.ServiceListItem(ctx, svc.Id)
	if err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	return s.serviceView(ctx, updated)
}

func (s Service) UpdateServiceBasic(ctx context.Context, userId string, serviceId string, input servicedto.ServiceBasicUpdateInput) (servicedto.ServiceView, error) {
	svc, err := s.serviceForUser(ctx, userId, serviceId)
	if err != nil {
		return servicedto.ServiceView{}, err
	}
	versionId := strings.TrimSpace(input.VersionId)
	instanceKey := strings.TrimSpace(input.InstanceKey)
	if versionId == "" || instanceKey == "" {
		return servicedto.ServiceView{}, apperror.New(apperror.KindValidation, "version_id and instance_key are required")
	}
	version, components, err := s.versionComponents(ctx, versionId, svc.ApplicationId)
	if err != nil {
		return servicedto.ServiceView{}, err
	}
	if existing, err := s.service.ServiceByKey(ctx, svc.ApplicationId, instanceKey); err == nil && existing.Id != svc.Id {
		return servicedto.ServiceView{}, apperror.New(apperror.KindConflict, "Service instance already exists")
	} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	exposes, err := s.service.ServiceExposesByService(ctx, svc.Id)
	if err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service exposes", err)
	}
	if err := validateRuntimeConfig(svc.RuntimeConfig, components); err != nil {
		return servicedto.ServiceView{}, err
	}
	if err := validateServiceExposes(exposes, components); err != nil {
		return servicedto.ServiceView{}, err
	}
	if err := s.validateExposeConflicts(ctx, svc.Id, exposes); err != nil {
		return servicedto.ServiceView{}, err
	}
	svc.VersionId = version.Id
	svc.InstanceKey = instanceKey
	if err := s.service.UpdateServiceConfiguration(ctx, svc, exposes); err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to update service basic information", err)
	}
	updated, err := s.service.ServiceListItem(ctx, svc.Id)
	if err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	return s.serviceView(ctx, updated)
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
	serviceId := strings.TrimSpace(input.ServiceId)
	if serviceId == "" {
		return model.Service{}, apperror.New(apperror.KindValidation, "service_id is required")
	}
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

func (s Service) serviceForUser(ctx context.Context, userId string, serviceId string) (model.Service, error) {
	serviceId = strings.TrimSpace(serviceId)
	if serviceId == "" {
		return model.Service{}, apperror.New(apperror.KindValidation, "service_id is required")
	}
	svc, err := s.service.Service(ctx, serviceId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Service{}, apperror.New(apperror.KindNotFound, "Service "+serviceId+" not found")
		}
		return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	if _, err := s.loadApplicationForUser(ctx, userId, svc.ApplicationId); err != nil {
		return model.Service{}, err
	}
	return svc, nil
}

func (s Service) versionComponents(ctx context.Context, versionId string, applicationId string) (model.Version, []model.VersionComponent, error) {
	version, err := s.application.Version(ctx, versionId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Version{}, nil, apperror.New(apperror.KindNotFound, "Version "+versionId+" not found")
		}
		return model.Version{}, nil, apperror.Wrap(apperror.KindInternal, "Failed to load version", err)
	}
	if version.ApplicationId != applicationId {
		return model.Version{}, nil, apperror.New(apperror.KindValidation, "Version does not belong to the service application")
	}
	components, err := s.application.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return model.Version{}, nil, apperror.Wrap(apperror.KindInternal, "Failed to load version components", err)
	}
	return version, components, nil
}

func normalizeRuntimeConfig(values map[string]string) (map[string]string, error) {
	result := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, apperror.New(apperror.KindValidation, "runtime config key is required")
		}
		if _, exists := result[key]; exists {
			return nil, apperror.New(apperror.KindValidation, "runtime config contains duplicate keys")
		}
		result[key] = value
	}
	return result, nil
}

func validateRuntimeConfig(values map[string]string, components []model.VersionComponent) error {
	missing, _ := runtimeconfig.Validate(values, components)
	if len(missing) > 0 {
		return apperror.New(apperror.KindValidation, fmt.Sprintf("missing runtime config keys: %s", strings.Join(missing, ", ")))
	}
	return nil
}

func optionalText(value *string) *string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	copy := strings.TrimSpace(*value)
	return &copy
}

func serviceExposesFromInputs(inputs []servicedto.ServiceExposeInput) ([]model.ServiceExpose, error) {
	result := make([]model.ServiceExpose, 0, len(inputs))
	for _, input := range inputs {
		componentName := strings.TrimSpace(input.ComponentName)
		protocol := strings.ToLower(strings.TrimSpace(input.Protocol))
		access := strings.ToLower(strings.TrimSpace(input.Access))
		if componentName == "" || (protocol != "http" && protocol != "tcp") || (access != "local" && access != "public") {
			return nil, apperror.New(apperror.KindValidation, "Invalid service expose fields")
		}
		if input.ContainerPort < 1 || input.ContainerPort > 65535 {
			return nil, apperror.New(apperror.KindValidation, "expose container_port out of range")
		}
		var listenPort *int
		if input.ListenPort != nil {
			if *input.ListenPort < 1 || *input.ListenPort > 65535 {
				return nil, apperror.New(apperror.KindValidation, "expose listen_port out of range")
			}
			value := *input.ListenPort
			listenPort = &value
		}
		pathPrefix := optionalText(input.PathPrefix)
		if protocol == "tcp" && pathPrefix != nil && *pathPrefix != "" {
			return nil, apperror.New(apperror.KindValidation, "path_prefix is only allowed for http expose")
		}
		result = append(result, model.ServiceExpose{
			ComponentName: componentName, Protocol: protocol, ContainerPort: input.ContainerPort,
			PathPrefix: pathPrefix, Access: access, ListenPort: listenPort,
		})
	}
	return result, nil
}

func validateServiceExposes(exposes []model.ServiceExpose, components []model.VersionComponent) error {
	componentNames := make(map[string]struct{}, len(components))
	for _, component := range components {
		componentNames[component.Name] = struct{}{}
	}
	seen := make(map[string]struct{}, len(exposes))
	localListens := map[int]struct{}{}
	httpPaths := map[string]struct{}{}
	for _, expose := range exposes {
		if _, ok := componentNames[expose.ComponentName]; !ok {
			return apperror.New(apperror.KindValidation, "Expose component is not present in the selected version")
		}
		key := fmt.Sprintf("%s|%s|%d", expose.ComponentName, expose.Protocol, expose.ContainerPort)
		if _, ok := seen[key]; ok {
			return apperror.New(apperror.KindValidation, "Duplicate service expose")
		}
		seen[key] = struct{}{}
		listen := expose.ContainerPort
		if expose.ListenPort != nil {
			listen = *expose.ListenPort
		}
		if expose.Access == "local" {
			if _, exists := localListens[listen]; exists {
				return apperror.New(apperror.KindValidation, "Duplicate local listen port")
			}
			localListens[listen] = struct{}{}
		}
		if expose.Access == "public" && expose.Protocol == "http" {
			path := ""
			if expose.PathPrefix != nil {
				path = *expose.PathPrefix
			}
			if _, exists := httpPaths[path]; exists {
				return apperror.New(apperror.KindValidation, "Duplicate public HTTP path prefix")
			}
			httpPaths[path] = struct{}{}
		}
	}
	return nil
}

func (s Service) validateExposeConflicts(ctx context.Context, serviceId string, exposes []model.ServiceExpose) error {
	for _, expose := range exposes {
		listen := expose.ContainerPort
		if expose.ListenPort != nil {
			listen = *expose.ListenPort
		}
		if expose.Access == "local" {
			existing, err := s.service.LocalServiceExposesByListen(ctx, listen)
			if err != nil {
				return apperror.Wrap(apperror.KindInternal, "Failed to check local listen conflicts", err)
			}
			for _, item := range existing {
				if item.ServiceId != serviceId {
					return apperror.New(apperror.KindConflict, "Local listen port is already used by another service")
				}
			}
		}
		if expose.Access == "public" && expose.Protocol == "tcp" {
			existing, err := s.service.PublicTCPServiceExposesByListen(ctx, listen)
			if err != nil {
				return apperror.Wrap(apperror.KindInternal, "Failed to check public TCP conflicts", err)
			}
			for _, item := range existing {
				if item.ServiceId != serviceId {
					return apperror.New(apperror.KindConflict, "Public TCP listen port is already used by another service")
				}
			}
		}
	}
	return nil
}

func (s Service) serviceView(ctx context.Context, item model.ServiceListItem) (servicedto.ServiceView, error) {
	exposes, err := s.service.ServiceExposesByService(ctx, item.Id)
	if err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service exposes", err)
	}
	view := serviceViewFromListItem(item)
	view.Exposes = exposes
	return view, nil
}

func serviceViewFromListItem(item model.ServiceListItem) servicedto.ServiceView {
	return servicedto.ServiceView{
		Service:         item.Service(),
		ApplicationName: item.ApplicationName,
		ApplicationCode: item.ApplicationCode,
		ApplicationKind: item.ApplicationKind,
		VersionLabel:    item.VersionLabel,
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
