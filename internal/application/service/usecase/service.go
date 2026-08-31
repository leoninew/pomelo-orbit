package servicesvc

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	deploymentsvc "github.com/leoninew/pomelo-orbit/internal/application/deployment/usecase"
	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
	"github.com/leoninew/pomelo-orbit/internal/common/commandline"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type applicationReader interface {
	Application(ctx context.Context, id string) (model.Application, error)
	Version(ctx context.Context, id string) (model.Version, error)
	VersionComponentsByVersion(ctx context.Context, versionId string) ([]model.VersionComponent, error)
}

type serviceStore interface{ repository.ServiceStore }

type Service struct {
	project     repository.ProjectReader
	application applicationReader
	service     serviceStore
	deployment  repository.DeploymentStore
}

var serviceEnvironmentKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func New(project repository.ProjectReader, application applicationReader, service serviceStore, deployment repository.DeploymentStore) Service {
	return Service{project: project, application: application, service: service, deployment: deployment}
}

func (s Service) DeleteService(ctx context.Context, userId, serviceId string) error {
	item, err := s.serviceForUser(ctx, userId, serviceId)
	if err != nil {
		return err
	}
	if item.Status != status.ServiceStatusStopped && item.Status != status.ServiceStatusFaulted {
		return apperror.New(apperror.KindValidation, "Only stopped or faulted services can be deleted")
	}
	if err := s.service.DeleteService(ctx, item.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete service", err)
	}
	return nil
}

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

func (s Service) GetService(ctx context.Context, userId, serviceId string) (servicedto.ServiceView, error) {
	item, err := s.service.ServiceListItem(ctx, strings.TrimSpace(serviceId))
	if err != nil {
		return servicedto.ServiceView{}, serviceReadError("Service", serviceId, err)
	}
	if _, err := s.loadApplicationForUser(ctx, userId, item.ApplicationId); err != nil {
		return servicedto.ServiceView{}, err
	}
	return s.serviceView(ctx, item)
}

func (s Service) CreateService(ctx context.Context, userId string, input servicedto.ServiceCreateInput) (servicedto.ServiceView, error) {
	applicationId, versionId, instanceKey := strings.TrimSpace(input.ApplicationId), strings.TrimSpace(input.VersionId), strings.TrimSpace(input.InstanceKey)
	code, err := normalizeServiceCode(input.Code)
	if err != nil {
		return servicedto.ServiceView{}, apperror.New(apperror.KindValidation, err.Error())
	}
	if applicationId == "" || versionId == "" || instanceKey == "" || code == "" {
		return servicedto.ServiceView{}, apperror.New(apperror.KindValidation, "application_id, version_id, instance_key and code are required")
	}
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return servicedto.ServiceView{}, err
	}
	version, declarations, err := s.versionComponents(ctx, versionId, app.Id)
	if err != nil {
		return servicedto.ServiceView{}, err
	}
	if _, err := s.service.ServiceByCode(ctx, code); err == nil {
		return servicedto.ServiceView{}, apperror.New(apperror.KindConflict, "Service code already exists")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service code", err)
	}
	if _, err := s.service.ServiceByKey(ctx, app.Id, instanceKey); err == nil {
		return servicedto.ServiceView{}, apperror.New(apperror.KindConflict, "Service instance already exists")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	svc := model.Service{Id: idutil.NewId(), ApplicationId: app.Id, InstanceKey: instanceKey, Code: code, VersionId: version.Id, Status: status.ServiceStatusStopped}
	components := mappedServiceComponents(svc.Id, declarations)
	if err := s.service.CreateServiceWithComponents(ctx, svc, components); err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to create service", err)
	}
	created, err := s.service.ServiceListItem(ctx, svc.Id)
	if err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	return s.serviceView(ctx, created)
}

func (s Service) UpdateServiceBasic(ctx context.Context, userId, serviceId string, input servicedto.ServiceBasicUpdateInput) (servicedto.ServiceView, error) {
	svc, err := s.serviceForUser(ctx, userId, serviceId)
	if err != nil {
		return servicedto.ServiceView{}, err
	}
	versionId, instanceKey := strings.TrimSpace(input.VersionId), strings.TrimSpace(input.InstanceKey)
	if versionId == "" || instanceKey == "" {
		return servicedto.ServiceView{}, apperror.New(apperror.KindValidation, "version_id and instance_key are required")
	}
	version, declarations, err := s.versionComponents(ctx, versionId, svc.ApplicationId)
	if err != nil {
		return servicedto.ServiceView{}, err
	}
	if existing, err := s.service.ServiceByKey(ctx, svc.ApplicationId, instanceKey); err == nil && existing.Id != svc.Id {
		return servicedto.ServiceView{}, apperror.New(apperror.KindConflict, "Service instance already exists")
	} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	mappings, err := s.service.ServiceComponentsByService(ctx, svc.Id)
	if err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service components", err)
	}
	if err := remapServiceComponents(mappings, declarations); err != nil {
		return servicedto.ServiceView{}, apperror.New(apperror.KindValidation, err.Error())
	}
	svc.VersionId, svc.InstanceKey = version.Id, instanceKey
	if err := s.service.UpdateServiceConfiguration(ctx, svc, mappings); err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to update service", err)
	}
	updated, err := s.service.ServiceListItem(ctx, svc.Id)
	if err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	return s.serviceView(ctx, updated)
}

func (s Service) GetServiceComponent(ctx context.Context, userId, serviceId, componentId string) (servicedto.ServiceComponentDetail, error) {
	svc, err := s.serviceForUser(ctx, userId, serviceId)
	if err != nil {
		return servicedto.ServiceComponentDetail{}, err
	}
	component, err := s.service.ServiceComponent(ctx, strings.TrimSpace(componentId))
	if err != nil {
		return servicedto.ServiceComponentDetail{}, serviceReadError("Service component", componentId, err)
	}
	if component.ServiceId != svc.Id {
		return servicedto.ServiceComponentDetail{}, apperror.New(apperror.KindNotFound, "Service component not found")
	}
	declarations, err := s.application.VersionComponentsByVersion(ctx, svc.VersionId)
	if err != nil {
		return servicedto.ServiceComponentDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load version components", err)
	}
	for _, declaration := range declarations {
		if declaration.Id == component.SourceVersionComponentId {
			return servicedto.ServiceComponentDetail{ServiceComponent: component, VersionComponent: declaration}, nil
		}
	}
	return servicedto.ServiceComponentDetail{}, apperror.New(apperror.KindValidation, "Service component declaration is no longer available")
}

func (s Service) UpdateServiceEnv(ctx context.Context, userId, serviceId string, input servicedto.ServiceEnvUpdateInput) (servicedto.ServiceView, error) {
	svc, err := s.serviceForUser(ctx, userId, serviceId)
	if err != nil {
		return servicedto.ServiceView{}, err
	}
	env, err := normalizeServiceEnv(input.Env)
	if err != nil {
		return servicedto.ServiceView{}, apperror.New(apperror.KindValidation, err.Error())
	}
	if err := s.service.ReplaceServiceEnv(ctx, svc.Id, env); err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to update service environment", err)
	}
	item, err := s.service.ServiceListItem(ctx, svc.Id)
	if err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	return s.serviceView(ctx, item)
}

func (s Service) UpdateServiceComponentOverlay(ctx context.Context, userId, serviceId, componentId string, input servicedto.ServiceComponentOverlayInput) (model.ServiceComponent, error) {
	detail, err := s.GetServiceComponent(ctx, userId, serviceId, componentId)
	if err != nil {
		return model.ServiceComponent{}, err
	}
	component := detail.ServiceComponent
	entrypoint, err := overlayCommand(input.Entrypoint)
	if err != nil {
		return model.ServiceComponent{}, apperror.New(apperror.KindValidation, err.Error())
	}
	command, err := overlayCommand(input.Command)
	if err != nil {
		return model.ServiceComponent{}, apperror.New(apperror.KindValidation, err.Error())
	}
	component.Entrypoint = entrypoint
	component.Command = command
	component.PullPolicy = input.PullPolicy
	component.RestartPolicy = input.RestartPolicy
	component.Env = append([]model.ServiceComponentEnv(nil), input.Env...)
	component.Mounts = append([]model.ServiceComponentMount(nil), input.Mounts...)
	component.Resources = input.Resources
	component.Endpoints = append([]model.ServiceComponentEndpoint(nil), input.Endpoints...)
	if err := normalizeOverlay(&component, detail.VersionComponent); err != nil {
		return model.ServiceComponent{}, apperror.New(apperror.KindValidation, err.Error())
	}
	if err := s.service.UpdateServiceComponentOverlay(ctx, component); err != nil {
		return model.ServiceComponent{}, apperror.Wrap(apperror.KindInternal, "Failed to update service component", err)
	}
	return s.service.ServiceComponent(ctx, component.Id)
}

func (s Service) ListServicesByApplication(ctx context.Context, userId, applicationId string) ([]model.Service, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return nil, err
	}
	items, err := s.service.ListServicesByApplication(ctx, app.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list services", err)
	}
	return items, nil
}

// ListServiceViewsByApplication provides the same runtime bindings as the
// application-scoped list, including active operation state for UI guards.
func (s Service) ListServiceViewsByApplication(ctx context.Context, userId, applicationId string) ([]servicedto.ServiceView, error) {
	items, err := s.ListServicesByApplication(ctx, userId, applicationId)
	if err != nil {
		return nil, err
	}
	views := make([]servicedto.ServiceView, 0, len(items))
	for _, item := range items {
		view := servicedto.ServiceView{Service: item}
		if s.deployment != nil {
			active, err := s.deployment.HasActiveDeployment(ctx, item.Id)
			if err != nil {
				return nil, apperror.Wrap(apperror.KindInternal, "Failed to load active deployment", err)
			}
			view.ActiveDeployment = active
		}
		views = append(views, view)
	}
	return views, nil
}

func (s Service) PrimaryServiceByApplication(ctx context.Context, userId, applicationId string) (*model.Service, error) {
	items, err := s.ListServicesByApplication(ctx, userId, applicationId)
	if err != nil || len(items) == 0 {
		return nil, err
	}
	return &items[0], nil
}
func (s Service) ResolveServiceTarget(ctx context.Context, userId, applicationId string, input servicedto.ServiceTargetInput) (model.Service, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return model.Service{}, err
	}
	svc, err := s.service.Service(ctx, strings.TrimSpace(input.ServiceId))
	if err != nil {
		return model.Service{}, serviceReadError("Service", input.ServiceId, err)
	}
	if svc.ApplicationId != app.Id {
		return model.Service{}, apperror.New(apperror.KindNotFound, "Service not found")
	}
	return svc, nil
}

func mappedServiceComponents(serviceId string, declarations []model.VersionComponent) []model.ServiceComponent {
	items := make([]model.ServiceComponent, 0, len(declarations))
	for _, declaration := range declarations {
		items = append(items, model.ServiceComponent{Id: idutil.NewId(), ServiceId: serviceId, SourceVersionComponentId: declaration.Id, ComponentName: declaration.Name, Status: "active"})
	}
	return items
}

// AlignComponentMappingsToVersion rewrites service_component rows for every
// Service bound to versionId so SourceVersionComponentId matches the current
// Version declarations. Gateway compile replaces version_component rows and
// ON DELETE CASCADE would otherwise leave Services without mappings.
func (s Service) AlignComponentMappingsToVersion(ctx context.Context, applicationId, versionId string) error {
	applicationId = strings.TrimSpace(applicationId)
	versionId = strings.TrimSpace(versionId)
	if applicationId == "" || versionId == "" {
		return apperror.New(apperror.KindValidation, "application_id and version_id are required")
	}
	version, err := s.application.Version(ctx, versionId)
	if err != nil {
		return serviceReadError("Version", versionId, err)
	}
	if version.ApplicationId != applicationId {
		return apperror.New(apperror.KindValidation, "version does not belong to application")
	}
	declarations, err := s.application.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load version components", err)
	}
	services, err := s.service.ListServicesByApplication(ctx, applicationId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to list services", err)
	}
	for _, svc := range services {
		if svc.VersionId != version.Id {
			continue
		}
		existing, err := s.service.ServiceComponentsByService(ctx, svc.Id)
		if err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to load service components", err)
		}
		aligned := alignServiceComponents(svc.Id, existing, declarations)
		if err := s.service.UpdateServiceConfiguration(ctx, svc, aligned); err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to align service component mappings", err)
		}
	}
	return nil
}

// alignServiceComponents keeps overlays by component name when possible and
// creates missing mappings after version_component rows are rewritten.
func alignServiceComponents(serviceId string, existing []model.ServiceComponent, declarations []model.VersionComponent) []model.ServiceComponent {
	byName := make(map[string]model.ServiceComponent, len(existing))
	for _, item := range existing {
		byName[item.ComponentName] = item
	}
	items := make([]model.ServiceComponent, 0, len(declarations))
	for _, declaration := range declarations {
		if previous, ok := byName[declaration.Name]; ok {
			previous.ServiceId = serviceId
			previous.SourceVersionComponentId = declaration.Id
			previous.ComponentName = declaration.Name
			if previous.Status == "" {
				previous.Status = "active"
			}
			items = append(items, previous)
			continue
		}
		items = append(items, model.ServiceComponent{
			Id: idutil.NewId(), ServiceId: serviceId, SourceVersionComponentId: declaration.Id,
			ComponentName: declaration.Name, Status: "active",
		})
	}
	return items
}

func remapServiceComponents(mappings []model.ServiceComponent, declarations []model.VersionComponent) error {
	if len(mappings) != len(declarations) {
		return fmt.Errorf("selected version has incompatible component declarations")
	}
	byName := make(map[string]model.VersionComponent, len(declarations))
	for _, declaration := range declarations {
		byName[declaration.Name] = declaration
	}
	for index := range mappings {
		declaration, ok := byName[mappings[index].ComponentName]
		if !ok {
			return fmt.Errorf("selected version does not declare component %s", mappings[index].ComponentName)
		}
		if err := normalizeOverlay(&mappings[index], declaration); err != nil {
			return fmt.Errorf("component %s is incompatible: %w", declaration.Name, err)
		}
		mappings[index].SourceVersionComponentId = declaration.Id
	}
	return nil
}

func normalizeOverlay(component *model.ServiceComponent, declaration model.VersionComponent) error {
	// An empty argv is an intentional override that restores the image default;
	// it must remain distinct from a nil value that inherits the declaration.
	if len(component.Entrypoint) > 0 && slices.Equal(component.Entrypoint, declaration.Entrypoint) {
		component.Entrypoint = nil
	}
	if len(component.Command) > 0 && slices.Equal(component.Command, declaration.Command) {
		component.Command = nil
	}
	if component.PullPolicy != nil {
		if !validServicePullPolicy(*component.PullPolicy) {
			return fmt.Errorf("pull_policy must be always, missing or never")
		}
		if *component.PullPolicy == declaration.PullPolicy {
			component.PullPolicy = nil
		}
	}
	if component.RestartPolicy != nil {
		if !validServiceRestartPolicy(*component.RestartPolicy) {
			return fmt.Errorf("restart_policy must be no, on-failure, always or unless-stopped")
		}
		if declaration.RestartPolicy != nil && *component.RestartPolicy == *declaration.RestartPolicy {
			component.RestartPolicy = nil
		}
	}
	allowedEnv := map[string]string{}
	for _, item := range declaration.Env {
		allowedEnv[item.Key] = item.Value
	}
	env := make([]model.ServiceComponentEnv, 0, len(component.Env))
	seenEnv := map[string]struct{}{}
	for _, item := range component.Env {
		if _, exists := seenEnv[item.Key]; exists {
			return fmt.Errorf("duplicate environment overlay %s", item.Key)
		}
		seenEnv[item.Key] = struct{}{}
		defaultValue, ok := allowedEnv[item.Key]
		if !ok {
			return fmt.Errorf("environment %s is not declared by the version", item.Key)
		}
		switch item.State {
		case model.ServiceComponentOverlayDeleted:
			if item.Value != nil {
				return fmt.Errorf("deleted environment %s cannot have a value", item.Key)
			}
			env = append(env, item)
		case model.ServiceComponentOverlayOverride:
			if item.Value == nil {
				return fmt.Errorf("environment %s requires a value", item.Key)
			}
			if *item.Value != defaultValue {
				env = append(env, item)
			}
		default:
			return fmt.Errorf("environment %s has invalid overlay state", item.Key)
		}
	}
	component.Env = env
	allowedMounts := make(map[string]model.VersionComponentMount, len(declaration.Mounts))
	for _, item := range declaration.Mounts {
		allowedMounts[item.Target] = item
	}
	mounts := make([]model.ServiceComponentMount, 0, len(component.Mounts))
	seenMount := map[string]struct{}{}
	for _, item := range component.Mounts {
		declarationMount, ok := allowedMounts[item.Target]
		if !ok {
			return fmt.Errorf("mount target %s is not declared by the version", item.Target)
		}
		if _, exists := seenMount[item.Target]; exists {
			return fmt.Errorf("duplicate mount overlay %s", item.Target)
		}
		seenMount[item.Target] = struct{}{}
		switch item.State {
		case model.ServiceComponentOverlayDeleted:
			if item.Source != nil || item.SourceIsHostPath != nil {
				return fmt.Errorf("mount %s is deleted and cannot have source settings", item.Target)
			}
			mounts = append(mounts, item)
		case model.ServiceComponentOverlayOverride:
			if item.Source == nil || item.SourceIsHostPath == nil {
				return fmt.Errorf("mount %s requires source and source_is_host_path", item.Target)
			}
			if err := model.ValidateMountSource(declarationMount.SourceType, *item.Source, *item.SourceIsHostPath); err != nil {
				return fmt.Errorf("component %s mount %s: %w", declaration.Name, item.Target, err)
			}
			if *item.Source != declarationMount.Source || *item.SourceIsHostPath != declarationMount.SourceIsHostPath {
				mounts = append(mounts, item)
			}
		default:
			return fmt.Errorf("mount has invalid overlay state")
		}
	}
	component.Mounts = mounts
	if component.Resources != nil {
		resource := component.Resources
		if declaration.Resources == nil {
			return fmt.Errorf("resources are not declared by the version")
		}
		if resource.State == model.ServiceComponentOverlayDeleted {
			if resource.LimitCPUs != nil || resource.LimitMemory != nil || resource.ReservationCPUs != nil || resource.ReservationMemory != nil {
				return fmt.Errorf("deleted resources cannot have values")
			}
		} else if resource.State != model.ServiceComponentOverlayOverride {
			return fmt.Errorf("resources have invalid overlay state")
		} else if resource.LimitCPUs == nil && resource.LimitMemory == nil && resource.ReservationCPUs == nil && resource.ReservationMemory == nil {
			component.Resources = nil
		}
	}
	allowedEndpoint := map[string]model.VersionComponentEndpoint{}
	for _, endpoint := range declaration.Endpoints {
		allowedEndpoint[model.EndpointDisplayName(endpoint.Protocol, endpoint.ContainerPort)] = endpoint
	}
	endpoints := make([]model.ServiceComponentEndpoint, 0, len(component.Endpoints))
	seenEndpoint := map[string]struct{}{}
	for _, item := range component.Endpoints {
		identity := model.EndpointDisplayName(item.Protocol, item.ContainerPort)
		base, ok := allowedEndpoint[identity]
		if !ok {
			return fmt.Errorf("endpoint %s is not declared by the version", identity)
		}
		if _, exists := seenEndpoint[identity]; exists {
			return fmt.Errorf("duplicate endpoint overlay %s", identity)
		}
		seenEndpoint[identity] = struct{}{}
		if item.State == model.ServiceComponentOverlayDeleted {
			if item.Mode != nil || item.BindAddress != nil || item.ListenPort != nil || item.Entrypoint != nil || item.PathPrefix != nil {
				return fmt.Errorf("deleted endpoint %s cannot have values", identity)
			}
			endpoints = append(endpoints, item)
			continue
		}
		if item.State != model.ServiceComponentOverlayOverride {
			return fmt.Errorf("endpoint %s has invalid overlay state", identity)
		}
		if item.Mode != nil {
			if *item.Mode != "internal" && *item.Mode != "local" && *item.Mode != "host" && *item.Mode != "gateway" {
				return fmt.Errorf("endpoint %s has invalid mode", identity)
			}
			if *item.Mode == "gateway" && base.Protocol != "http" {
				return fmt.Errorf("endpoint %s gateway mode requires http", identity)
			}
		}
		changed := (item.Mode != nil && *item.Mode != base.Mode) || (item.BindAddress != nil && (base.BindAddress == nil || *item.BindAddress != *base.BindAddress)) || (item.ListenPort != nil && (base.ListenPort == nil || *item.ListenPort != *base.ListenPort)) || (item.Entrypoint != nil && (base.Entrypoint == nil || *item.Entrypoint != *base.Entrypoint)) || (item.PathPrefix != nil && (base.PathPrefix == nil || *item.PathPrefix != *base.PathPrefix))
		if changed {
			endpoints = append(endpoints, item)
		}
	}
	component.Endpoints = endpoints
	return nil
}

func overlayCommand(input *string) ([]string, error) {
	if input == nil {
		return nil, nil
	}
	command, err := commandline.Parse(*input)
	if err != nil {
		return nil, fmt.Errorf("invalid component command")
	}
	return command, nil
}

func validServicePullPolicy(value string) bool {
	return value == "always" || value == "missing" || value == "never"
}

func validServiceRestartPolicy(value string) bool {
	return value == "no" || value == "on-failure" || value == "always" || value == "unless-stopped"
}

func normalizeServiceEnv(env []model.ServiceEnv) ([]model.ServiceEnv, error) {
	items := make([]model.ServiceEnv, 0, len(env))
	seen := make(map[string]struct{}, len(env))
	for _, item := range env {
		key := strings.TrimSpace(item.Key)
		if !serviceEnvironmentKey.MatchString(key) {
			return nil, fmt.Errorf("environment key %q is invalid", item.Key)
		}
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("duplicate service environment %s", key)
		}
		seen[key] = struct{}{}
		items = append(items, model.ServiceEnv{Key: key, Value: item.Value})
	}
	return items, nil
}

func normalizeServiceCode(input string) (string, error) {
	code := strings.TrimSpace(input)
	if code == "" {
		return "", nil
	}
	if !model.IsDNSLabel(code) {
		return "", fmt.Errorf("service code %q is invalid", code)
	}
	return code, nil
}

func (s Service) serviceView(ctx context.Context, item model.ServiceListItem) (servicedto.ServiceView, error) {
	env, err := s.service.ServiceEnvByService(ctx, item.Id)
	if err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service environment", err)
	}
	components, err := s.service.ServiceComponentsByService(ctx, item.Id)
	if err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service components", err)
	}
	view := servicedto.ServiceView{Service: item.Service(), Env: env, Components: components, ApplicationName: item.ApplicationName, ApplicationCode: item.ApplicationCode, ApplicationKind: item.ApplicationKind, VersionLabel: item.VersionLabel}
	if s.deployment == nil {
		return view, nil
	}
	active, err := s.deployment.HasActiveDeployment(ctx, item.Id)
	if err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load active deployment", err)
	}
	view.ActiveDeployment = active
	app, err := s.application.Application(ctx, item.ApplicationId)
	if err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load application plan context", err)
	}
	version, err := s.application.Version(ctx, item.VersionId)
	if err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load version plan context", err)
	}
	declarations, err := s.application.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load component plan context", err)
	}
	view.ComponentDefinitions = declarations
	plan, _, err := deploymentsvc.BuildEffectiveServicePlan(app, version, item.Service(), declarations, components, env, nil)
	if err != nil {
		view.PendingDeploy = true
		view.EffectiveError = err.Error()
		return view, nil
	}
	view.EffectiveComponents = plan.Components
	hash, err := deploymentsvc.EffectiveServicePlanHash(plan)
	if err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to hash service plan", err)
	}
	view.EffectivePlanHash = hash
	latest, err := s.deployment.LatestSuccessfulDeploymentPlanHash(ctx, item.Id)
	if err != nil {
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load deployed plan", err)
	}
	view.PendingDeploy = latest == nil || *latest != hash
	return view, nil
}

func (s Service) serviceForUser(ctx context.Context, userId, serviceId string) (model.Service, error) {
	svc, err := s.service.Service(ctx, strings.TrimSpace(serviceId))
	if err != nil {
		return model.Service{}, serviceReadError("Service", serviceId, err)
	}
	if _, err := s.loadApplicationForUser(ctx, userId, svc.ApplicationId); err != nil {
		return model.Service{}, err
	}
	return svc, nil
}
func (s Service) versionComponents(ctx context.Context, versionId, applicationId string) (model.Version, []model.VersionComponent, error) {
	version, err := s.application.Version(ctx, versionId)
	if err != nil {
		return model.Version{}, nil, serviceReadError("Version", versionId, err)
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
func (s Service) loadApplicationForUser(ctx context.Context, userId, applicationId string) (model.Application, error) {
	app, err := s.application.Application(ctx, strings.TrimSpace(applicationId))
	if err != nil {
		return model.Application{}, serviceReadError("Application", applicationId, err)
	}
	if app.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *app.ProjectId, userId); err != nil {
			return model.Application{}, err
		}
	}
	return app, nil
}
func (s Service) ensureProjectMembership(ctx context.Context, projectId, userId string) error {
	if _, err := s.project.Project(ctx, projectId); err != nil {
		return serviceReadError("Project", projectId, err)
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
func serviceReadError(kind, id string, err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return apperror.New(apperror.KindNotFound, kind+" "+id+" not found")
	}
	return apperror.Wrap(apperror.KindInternal, "Failed to load "+strings.ToLower(kind), err)
}
