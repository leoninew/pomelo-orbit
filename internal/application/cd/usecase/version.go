package cdsvc

import (
	"context"
	"errors"
	"strings"

	cdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func (s Service) ListVersions(ctx context.Context, userId string, applicationId string) ([]cdto.VersionView, error) {
	page, err := s.ListVersionsPage(ctx, userId, applicationId, 1, 100, "")
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

func (s Service) ListVersionsPage(ctx context.Context, userId string, applicationId string, page int, perPage int, search string) (repository.Page[cdto.VersionView], error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return repository.Page[cdto.VersionView]{}, err
	}
	versions, err := s.store.ListVersionsPage(ctx, app.Id, page, perPage, search)
	if err != nil {
		return repository.Page[cdto.VersionView]{}, apperror.Wrap(apperror.KindInternal, "Failed to list versions", err)
	}
	views := make([]cdto.VersionView, 0, len(versions.Items))
	for _, version := range versions.Items {
		view, err := s.versionView(ctx, version)
		if err != nil {
			return repository.Page[cdto.VersionView]{}, err
		}
		views = append(views, view)
	}
	return repository.Page[cdto.VersionView]{Items: views, Total: versions.Total, Page: versions.Page, PerPage: versions.PerPage}, nil
}

func (s Service) VersionForUser(ctx context.Context, userId string, versionId string) (cdto.VersionView, error) {
	version, err := s.loadVersionForUser(ctx, userId, versionId)
	if err != nil {
		return cdto.VersionView{}, err
	}
	return s.versionView(ctx, version)
}

func (s Service) CreateVersion(ctx context.Context, userId string, input cdto.VersionCreateInput) (cdto.VersionView, error) {
	app, err := s.loadApplicationForUser(ctx, userId, input.ApplicationId)
	if err != nil {
		return cdto.VersionView{}, err
	}
	label := strings.TrimSpace(input.Label)
	if label == "" || len(label) > 128 {
		return cdto.VersionView{}, apperror.New(apperror.KindValidation, "Invalid version label")
	}
	components, err := normalizeVersionComponents(input.Components)
	if err != nil {
		return cdto.VersionView{}, err
	}
	if err := validateVersionComponents(components); err != nil {
		return cdto.VersionView{}, apperror.New(apperror.KindValidation, err.Error())
	}
	exposes, err := normalizeVersionExposes(input.Exposes)
	if err != nil {
		return cdto.VersionView{}, err
	}
	if err := validateVersionExposes(exposes, components); err != nil {
		return cdto.VersionView{}, apperror.New(apperror.KindValidation, err.Error())
	}
	version := model.Version{
		Id:            idutil.NewId(),
		ApplicationId: app.Id,
		Label:         label,
		Status:        status.VersionStatusUnpublished,
		EnvJSON:       normalizeOptionalText(input.EnvJSON),
		Note:          normalizeOptionalText(input.Note),
	}
	for i := range components {
		components[i].Id = idutil.NewId()
		components[i].VersionId = version.Id
	}
	for i := range exposes {
		exposes[i].Id = idutil.NewId()
		exposes[i].VersionId = version.Id
	}
	if err := s.store.CreateVersionWithVersionComponentsAndExposes(ctx, version, components, exposes); err != nil {
		return cdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to create version", err)
	}
	return s.VersionForUser(ctx, userId, version.Id)
}

func (s Service) UpdateVersion(ctx context.Context, userId string, versionId string, input cdto.VersionUpdateInput) (cdto.VersionView, error) {
	version, err := s.loadVersionForUser(ctx, userId, versionId)
	if err != nil {
		return cdto.VersionView{}, err
	}
	if version.Status == status.VersionStatusPublished {
		return cdto.VersionView{}, apperror.New(apperror.KindValidation, "Published version is immutable")
	}
	if input.Label != nil {
		label := strings.TrimSpace(*input.Label)
		if label == "" || len(label) > 128 {
			return cdto.VersionView{}, apperror.New(apperror.KindValidation, "Invalid version label")
		}
		version.Label = label
	}
	if input.EnvJSON != nil {
		version.EnvJSON = normalizeOptionalText(input.EnvJSON)
	}
	if input.Note != nil {
		version.Note = normalizeOptionalText(input.Note)
	}
	if err := s.store.UpdateVersion(ctx, version); err != nil {
		return cdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to update version", err)
	}
	var components []model.VersionComponent
	if input.Components != nil {
		components, err = normalizeVersionComponents(*input.Components)
		if err != nil {
			return cdto.VersionView{}, err
		}
		if err := validateVersionComponents(components); err != nil {
			return cdto.VersionView{}, apperror.New(apperror.KindValidation, err.Error())
		}
		for i := range components {
			components[i].Id = idutil.NewId()
			components[i].VersionId = version.Id
		}
		if err := s.store.ReplaceVersionComponents(ctx, version.Id, components); err != nil {
			return cdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to replace components", err)
		}
	} else {
		components, err = s.store.VersionComponentsByVersion(ctx, version.Id)
		if err != nil {
			return cdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
		}
	}
	if input.Exposes != nil {
		exposes, err := normalizeVersionExposes(*input.Exposes)
		if err != nil {
			return cdto.VersionView{}, err
		}
		if err := validateVersionExposes(exposes, components); err != nil {
			return cdto.VersionView{}, apperror.New(apperror.KindValidation, err.Error())
		}
		for i := range exposes {
			exposes[i].Id = idutil.NewId()
			exposes[i].VersionId = version.Id
		}
		if err := s.store.ReplaceVersionExposes(ctx, version.Id, exposes); err != nil {
			return cdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to replace exposes", err)
		}
	}
	return s.VersionForUser(ctx, userId, version.Id)
}

func (s Service) PublishVersion(ctx context.Context, userId string, versionId string) (cdto.VersionView, error) {
	version, err := s.loadVersionForUser(ctx, userId, versionId)
	if err != nil {
		return cdto.VersionView{}, err
	}
	if version.Status == status.VersionStatusPublished {
		return s.VersionForUser(ctx, userId, version.Id)
	}
	components, err := s.store.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return cdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	if len(components) == 0 {
		return cdto.VersionView{}, apperror.New(apperror.KindValidation, "At least one component is required")
	}
	if err := validateVersionComponents(components); err != nil {
		return cdto.VersionView{}, apperror.New(apperror.KindValidation, err.Error())
	}
	exposes, err := s.store.VersionExposesByVersion(ctx, version.Id)
	if err != nil {
		return cdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to list exposes", err)
	}
	if err := validateVersionExposes(exposes, components); err != nil {
		return cdto.VersionView{}, apperror.New(apperror.KindValidation, err.Error())
	}
	version.Status = status.VersionStatusPublished
	if err := s.store.UpdateVersion(ctx, version); err != nil {
		return cdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to publish version", err)
	}
	return s.VersionForUser(ctx, userId, version.Id)
}

func (s Service) DeleteVersion(ctx context.Context, userId string, versionId string) error {
	version, err := s.loadVersionForUser(ctx, userId, versionId)
	if err != nil {
		return err
	}
	if version.Status != status.VersionStatusUnpublished {
		return apperror.New(apperror.KindValidation, "Only unpublished versions can be deleted")
	}
	refs, err := s.store.CountVersionRuntimeRefs(ctx, version.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check version references", err)
	}
	if refs > 0 {
		return apperror.New(apperror.KindValidation, "Version is referenced by a service or deployment")
	}
	if err := s.store.DeleteVersion(ctx, version.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete version", err)
	}
	return nil
}

func (s Service) ForkVersion(ctx context.Context, userId string, versionId string, label string) (cdto.VersionView, error) {
	source, err := s.loadVersionForUser(ctx, userId, versionId)
	if err != nil {
		return cdto.VersionView{}, err
	}
	label = strings.TrimSpace(label)
	if label == "" || len(label) > 128 {
		return cdto.VersionView{}, apperror.New(apperror.KindValidation, "Invalid version label")
	}
	components, err := s.store.VersionComponentsByVersion(ctx, source.Id)
	if err != nil {
		return cdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	exposes, err := s.store.VersionExposesByVersion(ctx, source.Id)
	if err != nil {
		return cdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to list exposes", err)
	}
	fromId := source.Id
	version := model.Version{
		Id:                   idutil.NewId(),
		ApplicationId:        source.ApplicationId,
		Label:                label,
		Status:               status.VersionStatusUnpublished,
		EnvJSON:              source.EnvJSON,
		CreatedFromVersionId: &fromId,
		Note:                 source.Note,
	}
	forkedComponents := make([]model.VersionComponent, 0, len(components))
	for _, component := range components {
		component.Id = idutil.NewId()
		component.VersionId = version.Id
		forkedComponents = append(forkedComponents, component)
	}
	forkedExposes := make([]model.VersionExpose, 0, len(exposes))
	for _, expose := range exposes {
		expose.Id = idutil.NewId()
		expose.VersionId = version.Id
		forkedExposes = append(forkedExposes, expose)
	}
	if err := s.store.CreateVersionWithVersionComponentsAndExposes(ctx, version, forkedComponents, forkedExposes); err != nil {
		return cdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to fork version", err)
	}
	return s.VersionForUser(ctx, userId, version.Id)
}

func (s Service) PreviewVersion(ctx context.Context, userId string, versionId string, environmentId string, instanceKey string) (string, error) {
	version, err := s.loadVersionForUser(ctx, userId, versionId)
	if err != nil {
		return "", err
	}
	app, err := s.store.Application(ctx, version.ApplicationId)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	environmentId = strings.TrimSpace(environmentId)
	if environmentId == "" {
		return "", apperror.New(apperror.KindValidation, "environment_id is required")
	}
	env, err := s.store.Environment(ctx, environmentId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", apperror.New(apperror.KindNotFound, "Environment not found")
		}
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load environment", err)
	}
	if app.ProjectId == nil || *app.ProjectId != env.ProjectId {
		return "", apperror.New(apperror.KindValidation, "application and environment must belong to the same project")
	}
	instanceKey, err = normalizeInstanceKey(instanceKey)
	if err != nil {
		return "", err
	}
	components, err := s.store.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	exposes, err := s.store.VersionExposesByVersion(ctx, version.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to list exposes", err)
	}
	physicalDir, err := s.workspace.PhysicalServiceDir(ctx, app.Code, env.Code, instanceKey)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to resolve physical service dir", err)
	}
	gateway, err := s.optionalGatewayForRender(ctx, components, exposes, app)
	if err != nil {
		return "", err
	}
	content, err := s.RenderCompose(ctx, RenderInput{
		App: app, Version: version, Components: components, Exposes: exposes,
		Env:            env,
		Service:        model.Service{InstanceKey: instanceKey},
		Gateway:        gateway,
		PhysicalSvcDir: physicalDir,
	})
	if err != nil {
		return "", apperror.New(apperror.KindValidation, err.Error())
	}
	return content, nil
}

// optionalGatewayForRender loads gateway config when Host/dashboard domains are needed.
func (s Service) optionalGatewayForRender(ctx context.Context, components []model.VersionComponent, exposes []model.VersionExpose, app model.Application) (*model.GatewayConfig, error) {
	kind := strings.TrimSpace(app.Kind)
	needsGateway := kind == status.ApplicationKindGateway || needsPublicGateway(exposes)
	if !needsGateway {
		return nil, nil
	}
	reader := s.gatewayConfigReader()
	if reader == nil {
		return nil, apperror.New(apperror.KindInternal, "gateway config store is not available")
	}
	// Prefer the gateway app's own config when rendering kind=gateway.
	if kind == status.ApplicationKindGateway {
		cfg, err := reader.GatewayConfig(ctx, app.Id)
		if err == nil {
			return &cfg, nil
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.Wrap(apperror.KindInternal, "Failed to load gateway config", err)
		}
	}
	return s.resolveGatewayForRender(ctx)
}

func (s Service) ListServicesByApplication(ctx context.Context, userId string, applicationId string) ([]model.Service, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return nil, err
	}
	services, err := s.store.ListServicesByApplication(ctx, app.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list services", err)
	}
	return services, nil
}

// ListServices returns runtime bindings (application + version + environment) for a project.
func (s Service) ListServices(ctx context.Context, userId string, input cdto.ServiceListInput) (repository.Page[cdto.ServiceView], error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return repository.Page[cdto.ServiceView]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[cdto.ServiceView]{}, err
	}
	page, err := s.store.ListServicesByProject(ctx, projectId, input.ApplicationId, input.EnvironmentId, input.Status, input.Search, input.Page, input.PerPage)
	if err != nil {
		return repository.Page[cdto.ServiceView]{}, apperror.Wrap(apperror.KindInternal, "Failed to list services", err)
	}
	items := make([]cdto.ServiceView, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, serviceViewFromListItem(item))
	}
	return repository.Page[cdto.ServiceView]{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage}, nil
}

// GetService returns one runtime binding with display labels when the user can access its application.
func (s Service) GetService(ctx context.Context, userId string, serviceId string) (cdto.ServiceView, error) {
	serviceId = strings.TrimSpace(serviceId)
	if serviceId == "" {
		return cdto.ServiceView{}, apperror.New(apperror.KindValidation, "service_id is required")
	}
	item, err := s.store.ServiceListItem(ctx, serviceId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return cdto.ServiceView{}, apperror.New(apperror.KindNotFound, "Service "+serviceId+" not found")
		}
		return cdto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	if _, err := s.loadApplicationForUser(ctx, userId, item.ApplicationId); err != nil {
		return cdto.ServiceView{}, err
	}
	return serviceViewFromListItem(item), nil
}

func serviceViewFromListItem(item model.ServiceListItem) cdto.ServiceView {
	return cdto.ServiceView{
		Service:                    item.Service(),
		ApplicationName:            item.ApplicationName,
		ApplicationCode:            item.ApplicationCode,
		ApplicationKind:            item.ApplicationKind,
		EnvironmentName:            item.EnvironmentName,
		EnvironmentCode:            item.EnvironmentCode,
		VersionLabel:               item.VersionLabel,
		LastSuccessfulVersionLabel: item.LastSuccessfulVersionLabel,
	}
}

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

func (s Service) versionView(ctx context.Context, version model.Version) (cdto.VersionView, error) {
	components, err := s.store.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return cdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	exposes, err := s.store.VersionExposesByVersion(ctx, version.Id)
	if err != nil {
		return cdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to list exposes", err)
	}
	return cdto.VersionView{Version: version, Components: components, Exposes: exposes}, nil
}

func (s Service) loadVersionForUser(ctx context.Context, userId string, versionId string) (model.Version, error) {
	versionId = strings.TrimSpace(versionId)
	version, err := s.store.Version(ctx, versionId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Version{}, apperror.New(apperror.KindNotFound, "Version "+versionId+" not found")
		}
		return model.Version{}, apperror.Wrap(apperror.KindInternal, "Failed to load version", err)
	}
	if _, err := s.loadApplicationForUser(ctx, userId, version.ApplicationId); err != nil {
		return model.Version{}, err
	}
	return version, nil
}

func normalizeVersionComponents(inputs []cdto.VersionComponentInput) ([]model.VersionComponent, error) {
	// Empty is allowed for unpublished drafts; PublishVersion enforces at least one component.
	if len(inputs) == 0 {
		return []model.VersionComponent{}, nil
	}
	components := make([]model.VersionComponent, 0, len(inputs))
	for _, input := range inputs {
		name := strings.TrimSpace(input.Name)
		image := strings.TrimSpace(input.Image)
		if name == "" || image == "" {
			return nil, apperror.New(apperror.KindValidation, "Component name and image are required")
		}
		if !applicationCreateCodePattern.MatchString(name) {
			return nil, apperror.New(apperror.KindValidation, "Component name must match ^[a-z][a-z0-9-]*$")
		}
		components = append(components, model.VersionComponent{
			Name:            name,
			Image:           image,
			CommandJSON:     normalizeOptionalText(input.CommandJSON),
			ArgsJSON:        normalizeOptionalText(input.ArgsJSON),
			EnvJSON:         normalizeOptionalText(input.EnvJSON),
			PortsJSON:       normalizeOptionalText(input.PortsJSON),
			MountsJSON:      normalizeOptionalText(input.MountsJSON),
			NetworksJSON:    normalizeOptionalText(input.NetworksJSON),
			DependsOnJSON:   normalizeOptionalText(input.DependsOnJSON),
			HealthcheckJSON: normalizeOptionalText(input.HealthcheckJSON),
			ResourcesJSON:   normalizeOptionalText(input.ResourcesJSON),
			PullPolicy:      normalizeOptionalText(input.PullPolicy),
		})
	}
	return components, nil
}

func normalizeVersionExposes(inputs []cdto.VersionExposeInput) ([]model.VersionExpose, error) {
	exposes := make([]model.VersionExpose, 0, len(inputs))
	for _, input := range inputs {
		protocol := strings.ToLower(strings.TrimSpace(input.Protocol))
		componentName := strings.TrimSpace(input.ComponentName)
		if componentName == "" || (protocol != "http" && protocol != "tcp") || input.ContainerPort < 1 || input.ContainerPort > 65535 {
			return nil, apperror.New(apperror.KindValidation, "Invalid expose fields")
		}
		access := strings.ToLower(strings.TrimSpace(input.Access))
		if access == "" {
			access = exposeAccessPublic
		}
		if access != exposeAccessLocal && access != exposeAccessPublic {
			return nil, apperror.New(apperror.KindValidation, "expose access must be local or public")
		}
		var listenPort *int
		if input.ListenPort != nil && *input.ListenPort > 0 {
			if *input.ListenPort > 65535 {
				return nil, apperror.New(apperror.KindValidation, "expose listen_port out of range")
			}
			v := *input.ListenPort
			listenPort = &v
		}
		pathPrefix := normalizeOptionalText(input.PathPrefix)
		if protocol == "tcp" && pathPrefix != nil && strings.TrimSpace(*pathPrefix) != "" {
			return nil, apperror.New(apperror.KindValidation, "path_prefix is only allowed for http expose")
		}
		exposes = append(exposes, model.VersionExpose{
			ComponentName: componentName,
			Protocol:      protocol,
			ContainerPort: input.ContainerPort,
			PathPrefix:    pathPrefix,
			Access:        access,
			ListenPort:    listenPort,
		})
	}
	return exposes, nil
}

func normalizeInstanceKey(instanceKey string) (string, error) {
	instanceKey = strings.TrimSpace(instanceKey)
	if instanceKey == "" {
		instanceKey = "default"
	}
	if instanceKey != "default" {
		return "", apperror.New(apperror.KindValidation, "instance_key must be default (single runtime per application)")
	}
	return instanceKey, nil
}
