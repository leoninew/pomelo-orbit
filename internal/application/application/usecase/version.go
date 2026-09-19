package applicationsvc

import (
	"context"
	"errors"
	"strings"

	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
	"github.com/leoninew/pomelo-orbit/internal/common/commandline"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func optionalText(value *string) *string {
	if value == nil {
		return nil
	}
	text := *value
	return &text
}

func (s Service) ListVersions(ctx context.Context, userId string, projectId string, applicationId string) ([]applicationdto.VersionView, error) {
	app, err := s.loadApplicationForUser(ctx, userId, projectId, applicationId)
	if err != nil {
		return nil, err
	}
	versions, err := s.store.ListVersions(ctx, projectId, app.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list versions", err)
	}
	views := make([]applicationdto.VersionView, 0, len(versions))
	for _, version := range versions {
		view, err := s.versionView(ctx, projectId, version)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func (s Service) ListVersionsPage(ctx context.Context, userId string, projectId string, applicationId string, page int, perPage int, search string) (repository.Page[applicationdto.VersionView], error) {
	app, err := s.loadApplicationForUser(ctx, userId, projectId, applicationId)
	if err != nil {
		return repository.Page[applicationdto.VersionView]{}, err
	}
	versions, err := s.store.ListVersionsPage(ctx, projectId, app.Id, page, perPage, search)
	if err != nil {
		return repository.Page[applicationdto.VersionView]{}, apperror.Wrap(apperror.KindInternal, "Failed to list versions", err)
	}
	views := make([]applicationdto.VersionView, 0, len(versions.Items))
	for _, version := range versions.Items {
		views = append(views, applicationdto.VersionView{Version: version})
	}
	return repository.Page[applicationdto.VersionView]{Items: views, Total: versions.Total, Page: versions.Page, PerPage: versions.PerPage}, nil
}

func (s Service) VersionForUser(ctx context.Context, userId string, projectId string, versionId string) (applicationdto.VersionView, error) {
	version, err := s.loadVersionForUser(ctx, userId, projectId, versionId)
	if err != nil {
		return applicationdto.VersionView{}, err
	}
	return s.versionView(ctx, projectId, version)
}

func (s Service) CreateVersion(ctx context.Context, userId string, projectId string, input applicationdto.VersionCreateInput) (applicationdto.VersionView, error) {
	app, err := s.loadApplicationForUser(ctx, userId, projectId, input.ApplicationId)
	if err != nil {
		return applicationdto.VersionView{}, err
	}
	label := input.Label
	if strings.TrimSpace(label) == "" || len(label) > 128 {
		return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, "Invalid version label")
	}
	components, err := versionComponentsFromInputs(input.Components)
	if err != nil {
		return applicationdto.VersionView{}, err
	}
	if err := validateVersionComponents(components); err != nil {
		return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, err.Error())
	}
	version := model.Version{
		Id:            idutil.NewId(),
		ApplicationId: app.Id,
		Label:         label,
		Status:        status.VersionStatusUnpublished,
		Note:          optionalText(input.Note),
	}
	for i := range components {
		components[i].Id = idutil.NewId()
		components[i].VersionId = version.Id
	}
	if err := s.store.CreateVersionWithVersionComponents(ctx, projectId, version, components); err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to create version", err)
	}
	return s.VersionForUser(ctx, userId, projectId, version.Id)
}

// CreateVersionFromDefinition creates an unpublished Version from its parsed
// component specification. The caller may choose an existing parent Version
// from the same Application to establish Version lineage.
func (s Service) CreateVersionFromDefinition(ctx context.Context, userId string, projectId string, applicationId string, input applicationdto.VersionDefinitionInput) (applicationdto.VersionView, error) {
	app, err := s.loadApplicationForUser(ctx, userId, projectId, applicationId)
	if err != nil {
		return applicationdto.VersionView{}, err
	}
	label := strings.TrimSpace(input.Label)
	if label == "" || len(label) > 128 {
		return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, "Invalid version label")
	}
	var createdFromVersionId *string
	if input.CreatedFromVersionId != nil {
		value := strings.TrimSpace(*input.CreatedFromVersionId)
		if value != "" {
			createdFromVersionId = &value
		}
	}
	if createdFromVersionId != nil {
		parent, err := s.store.Version(ctx, projectId, *createdFromVersionId)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return applicationdto.VersionView{}, apperror.New(apperror.KindNotFound, "Version "+*createdFromVersionId+" not found")
			}
			return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to load parent version", err)
		}
		if parent.ApplicationId != app.Id {
			return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, "Parent version does not belong to the application")
		}
	}
	components := cloneVersionDefinitionComponents(input.Components)
	if err := validateVersionDefinitionComponents(components); err != nil {
		return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, err.Error())
	}
	version := model.Version{
		Id:                   idutil.NewId(),
		ApplicationId:        app.Id,
		Label:                label,
		Status:               status.VersionStatusUnpublished,
		CreatedFromVersionId: createdFromVersionId,
		Note:                 optionalText(input.Note),
	}
	for index := range components {
		components[index].Id = idutil.NewId()
		components[index].VersionId = version.Id
	}
	if err := s.store.CreateVersionWithVersionComponents(ctx, projectId, version, components); err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to create version", err)
	}
	return s.VersionForUser(ctx, userId, projectId, version.Id)
}

func (s Service) createDefinitionVersions(ctx context.Context, projectId string, app model.Application, definitions []applicationdto.VersionDefinition) ([]applicationdto.VersionDefinition, error) {
	if len(definitions) == 0 {
		return nil, apperror.New(apperror.KindValidation, "Application definition requires at least one Version")
	}
	bySourceId := make(map[string]applicationdto.VersionDefinition, len(definitions))
	for _, definition := range definitions {
		sourceId := strings.TrimSpace(definition.Version.Id)
		if sourceId == "" {
			return nil, apperror.New(apperror.KindValidation, "Application definition Version id is required")
		}
		if _, exists := bySourceId[sourceId]; exists {
			return nil, apperror.New(apperror.KindValidation, "Application definition has duplicate Version id")
		}
		bySourceId[sourceId] = definition
	}
	createdBySourceId := make(map[string]applicationdto.VersionDefinition, len(definitions))
	visiting := make(map[string]bool, len(definitions))
	var create func(string) error
	create = func(sourceId string) error {
		if _, exists := createdBySourceId[sourceId]; exists {
			return nil
		}
		if visiting[sourceId] {
			return apperror.New(apperror.KindValidation, "Application definition Version lineage contains a cycle")
		}
		definition, exists := bySourceId[sourceId]
		if !exists {
			return apperror.New(apperror.KindValidation, "Application definition Version lineage references an unknown Version")
		}
		visiting[sourceId] = true
		var parentId *string
		if definition.Version.CreatedFromVersionId != nil {
			parentSourceId := strings.TrimSpace(*definition.Version.CreatedFromVersionId)
			if parentSourceId == "" {
				return apperror.New(apperror.KindValidation, "Application definition Version parent id is invalid")
			}
			if err := create(parentSourceId); err != nil {
				return err
			}
			parent := createdBySourceId[parentSourceId].Version.Id
			parentId = &parent
		}
		components := cloneVersionDefinitionComponents(definition.Components)
		for index := range components {
			components[index].ArtifactId = nil
			components[index].Artifact = nil
		}
		if err := validateVersionDefinitionComponents(components); err != nil {
			return apperror.New(apperror.KindValidation, err.Error())
		}
		label := strings.TrimSpace(definition.Version.Label)
		if label == "" || len(label) > 128 {
			return apperror.New(apperror.KindValidation, "Invalid version label")
		}
		if definition.Version.Status != status.VersionStatusUnpublished && definition.Version.Status != status.VersionStatusPublished {
			return apperror.New(apperror.KindValidation, "Invalid version status")
		}
		version := model.Version{
			Id: idutil.NewId(), ApplicationId: app.Id, Label: label, Status: definition.Version.Status,
			CreatedFromVersionId: parentId, Note: optionalText(definition.Version.Note),
			ComponentSummary: model.VersionComponentSummary(components),
		}
		for index := range components {
			components[index].Id = idutil.NewId()
			components[index].VersionId = version.Id
		}
		if err := s.store.CreateVersionWithVersionComponents(ctx, projectId, version, components); err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to create Version definition", err)
		}
		createdBySourceId[sourceId] = applicationdto.VersionDefinition{Version: version, Components: components}
		visiting[sourceId] = false
		return nil
	}
	for _, definition := range definitions {
		if err := create(strings.TrimSpace(definition.Version.Id)); err != nil {
			return nil, err
		}
	}
	created := make([]applicationdto.VersionDefinition, 0, len(definitions))
	for _, definition := range definitions {
		created = append(created, createdBySourceId[strings.TrimSpace(definition.Version.Id)])
	}
	return created, nil
}

func (s Service) UpdateVersion(ctx context.Context, userId string, projectId string, versionId string, input applicationdto.VersionUpdateInput) (applicationdto.VersionView, error) {
	version, err := s.loadVersionForUser(ctx, userId, projectId, versionId)
	if err != nil {
		return applicationdto.VersionView{}, err
	}
	if input.Label != nil {
		label := *input.Label
		if strings.TrimSpace(label) == "" || len(label) > 128 {
			return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, "Invalid version label")
		}
		version.Label = label
	}
	if input.Note != nil {
		version.Note = optionalText(input.Note)
	}
	if err := s.store.UpdateVersion(ctx, projectId, version); err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to update version", err)
	}
	return s.VersionForUser(ctx, userId, projectId, version.Id)
}

func (s Service) VersionComponentForUser(ctx context.Context, userId string, projectId string, versionId string, componentId string) (model.VersionComponent, error) {
	version, err := s.loadVersionForUser(ctx, userId, projectId, versionId)
	if err != nil {
		return model.VersionComponent{}, err
	}
	component, err := s.store.VersionComponent(ctx, projectId, strings.TrimSpace(componentId))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.VersionComponent{}, apperror.New(apperror.KindNotFound, "Component "+componentId+" not found")
		}
		return model.VersionComponent{}, apperror.Wrap(apperror.KindInternal, "Failed to load component", err)
	}
	if component.VersionId != version.Id {
		return model.VersionComponent{}, apperror.New(apperror.KindNotFound, "Component "+componentId+" not found")
	}
	return component, nil
}

func (s Service) CreateVersionComponent(ctx context.Context, userId string, projectId string, versionId string, input applicationdto.VersionComponentInput) (model.VersionComponent, error) {
	version, err := s.loadVersionForUser(ctx, userId, projectId, versionId)
	if err != nil {
		return model.VersionComponent{}, err
	}
	if version.Status != status.VersionStatusUnpublished {
		return model.VersionComponent{}, apperror.New(apperror.KindValidation, "Published version components cannot be changed")
	}
	components, err := s.store.VersionComponentsByVersion(ctx, projectId, version.Id)
	if err != nil {
		return model.VersionComponent{}, apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	componentsFromInput, err := versionComponentsFromInputs([]applicationdto.VersionComponentInput{input})
	if err != nil {
		return model.VersionComponent{}, err
	}
	component := componentsFromInput[0]
	component.Id = idutil.NewId()
	component.VersionId = version.Id
	components = append(components, component)
	if err := validateVersionComponents(components); err != nil {
		return model.VersionComponent{}, apperror.New(apperror.KindValidation, err.Error())
	}
	if err := s.store.CreateVersionComponent(ctx, projectId, component); err != nil {
		return model.VersionComponent{}, apperror.Wrap(apperror.KindInternal, "Failed to create component", err)
	}
	return s.VersionComponentForUser(ctx, userId, projectId, version.Id, component.Id)
}

func (s Service) UpdateVersionComponentBasic(ctx context.Context, userId string, projectId string, versionId string, componentId string, input applicationdto.VersionComponentBasicUpdateInput) (model.VersionComponent, error) {
	entrypoint, err := parseComponentCommand(input.Entrypoint)
	if err != nil {
		return model.VersionComponent{}, err
	}
	command, err := parseComponentCommand(input.Command)
	if err != nil {
		return model.VersionComponent{}, err
	}
	return s.updateVersionComponentGroup(ctx, userId, projectId, versionId, componentId, func(component *model.VersionComponent) {
		component.Name = input.Name
		component.Image = input.Image
		component.Entrypoint = entrypoint
		component.Command = command
		component.PullPolicy = input.PullPolicy
		component.RestartPolicy = input.RestartPolicy
	}, func(ctx context.Context, projectId string, component model.VersionComponent, oldName string) error {
		return s.store.UpdateVersionComponentBasic(ctx, projectId, component, oldName)
	})
}

func (s Service) UpdateVersionComponentRuntime(ctx context.Context, userId string, projectId string, versionId string, componentId string, input applicationdto.VersionComponentRuntimeUpdateInput) (model.VersionComponent, error) {
	healthcheck, err := componentHealthcheckFromInput(input.Healthcheck)
	if err != nil {
		return model.VersionComponent{}, err
	}
	return s.updateVersionComponentGroup(ctx, userId, projectId, versionId, componentId, func(component *model.VersionComponent) {
		component.Healthcheck = healthcheck
	}, func(ctx context.Context, projectId string, component model.VersionComponent, _ string) error {
		return s.store.UpdateVersionComponentRuntime(ctx, projectId, component)
	})
}

func (s Service) UpdateVersionComponentEndpoints(ctx context.Context, userId string, projectId string, versionId string, componentId string, input applicationdto.VersionComponentEndpointsUpdateInput) (model.VersionComponent, error) {
	return s.updateVersionComponentGroup(ctx, userId, projectId, versionId, componentId, func(component *model.VersionComponent) {
		component.Endpoints = append([]model.VersionComponentEndpoint(nil), input.Endpoints...)
	}, func(ctx context.Context, projectId string, component model.VersionComponent, _ string) error {
		return s.store.UpdateVersionComponentEndpoints(ctx, projectId, component)
	})
}

func (s Service) UpdateVersionComponentEnv(ctx context.Context, userId string, projectId string, versionId string, componentId string, input applicationdto.VersionComponentEnvUpdateInput) (model.VersionComponent, error) {
	return s.updateVersionComponentGroup(ctx, userId, projectId, versionId, componentId, func(component *model.VersionComponent) {
		component.Env = append([]model.VersionComponentEnv(nil), input.Env...)
	}, func(ctx context.Context, projectId string, component model.VersionComponent, _ string) error {
		return s.store.UpdateVersionComponentEnv(ctx, projectId, component)
	})
}

func (s Service) UpdateVersionComponentMounts(ctx context.Context, userId string, projectId string, versionId string, componentId string, input applicationdto.VersionComponentMountsUpdateInput) (model.VersionComponent, error) {
	return s.updateVersionComponentGroup(ctx, userId, projectId, versionId, componentId, func(component *model.VersionComponent) {
		component.Mounts = append([]model.VersionComponentMount(nil), input.Mounts...)
	}, func(ctx context.Context, projectId string, component model.VersionComponent, _ string) error {
		return s.store.UpdateVersionComponentMounts(ctx, projectId, component)
	})
}

func (s Service) UpdateVersionComponentDependencies(ctx context.Context, userId string, projectId string, versionId string, componentId string, input applicationdto.VersionComponentDependenciesUpdateInput) (model.VersionComponent, error) {
	return s.updateVersionComponentGroup(ctx, userId, projectId, versionId, componentId, func(component *model.VersionComponent) {
		component.Dependencies = append([]model.VersionComponentDependency(nil), input.Dependencies...)
	}, func(ctx context.Context, projectId string, component model.VersionComponent, _ string) error {
		return s.store.UpdateVersionComponentDependencies(ctx, projectId, component)
	})
}

func (s Service) UpdateVersionComponentAdvanced(ctx context.Context, userId string, projectId string, versionId string, componentId string, input applicationdto.VersionComponentAdvancedUpdateInput) (model.VersionComponent, error) {
	return s.updateVersionComponentGroup(ctx, userId, projectId, versionId, componentId, func(component *model.VersionComponent) {
		component.Resources = cloneComponentResources(input.Resources)
		component.Tmpfs = append([]model.VersionComponentTmpfs(nil), input.Tmpfs...)
		component.Ulimits = append([]model.VersionComponentUlimit(nil), input.Ulimits...)
	}, func(ctx context.Context, projectId string, component model.VersionComponent, _ string) error {
		return s.store.UpdateVersionComponentAdvanced(ctx, projectId, component)
	})
}

func (s Service) UpdateVersionComponentDevices(ctx context.Context, userId string, projectId string, versionId string, componentId string, input applicationdto.VersionComponentDevicesUpdateInput) (model.VersionComponent, error) {
	return s.updateVersionComponentGroup(ctx, userId, projectId, versionId, componentId, func(component *model.VersionComponent) {
		component.Devices = cloneComponentDeviceRequests(input.Devices)
	}, func(ctx context.Context, projectId string, component model.VersionComponent, _ string) error {
		return s.store.UpdateVersionComponentDevices(ctx, projectId, component)
	})
}

func (s Service) updateVersionComponentGroup(ctx context.Context, userId string, projectId string, versionId string, componentId string, update func(*model.VersionComponent), persist func(context.Context, string, model.VersionComponent, string) error) (model.VersionComponent, error) {
	version, err := s.loadVersionForUser(ctx, userId, projectId, versionId)
	if err != nil {
		return model.VersionComponent{}, err
	}
	if version.Status != status.VersionStatusUnpublished {
		return model.VersionComponent{}, apperror.New(apperror.KindValidation, "Published version components cannot be changed")
	}
	existing, err := s.VersionComponentForUser(ctx, userId, projectId, version.Id, componentId)
	if err != nil {
		return model.VersionComponent{}, err
	}
	components, err := s.store.VersionComponentsByVersion(ctx, projectId, version.Id)
	if err != nil {
		return model.VersionComponent{}, apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	componentIndex := -1
	for index := range components {
		if components[index].Id == existing.Id {
			componentIndex = index
			break
		}
	}
	if componentIndex == -1 {
		return model.VersionComponent{}, apperror.New(apperror.KindInternal, "Component was not found in its version")
	}
	update(&components[componentIndex])
	component := components[componentIndex]
	if existing.Name != component.Name {
		for componentIndex := range components {
			for dependencyIndex := range components[componentIndex].Dependencies {
				if components[componentIndex].Dependencies[dependencyIndex].Name == existing.Name {
					components[componentIndex].Dependencies[dependencyIndex].Name = component.Name
				}
			}
		}
	}
	if err := validateVersionComponents(components); err != nil {
		return model.VersionComponent{}, apperror.New(apperror.KindValidation, err.Error())
	}
	if err := persist(ctx, projectId, component, existing.Name); err != nil {
		return model.VersionComponent{}, apperror.Wrap(apperror.KindInternal, "Failed to update component", err)
	}
	return s.VersionComponentForUser(ctx, userId, projectId, version.Id, component.Id)
}

func (s Service) DeleteVersionComponent(ctx context.Context, userId string, projectId string, versionId string, componentId string) error {
	version, err := s.loadVersionForUser(ctx, userId, projectId, versionId)
	if err != nil {
		return err
	}
	if version.Status != status.VersionStatusUnpublished {
		return apperror.New(apperror.KindValidation, "Published version components cannot be changed")
	}
	component, err := s.VersionComponentForUser(ctx, userId, projectId, version.Id, componentId)
	if err != nil {
		return err
	}
	components, err := s.store.VersionComponentsByVersion(ctx, projectId, version.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	references := make([]string, 0)
	for _, candidate := range components {
		for _, dependency := range candidate.Dependencies {
			if dependency.Name == component.Name {
				references = append(references, "component "+candidate.Name)
			}
		}
	}
	if len(references) > 0 {
		return apperror.New(apperror.KindValidation, "Component is referenced by "+strings.Join(references, ", "))
	}
	if err := s.store.DeleteVersionComponent(ctx, projectId, component); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete component", err)
	}
	return nil
}

func (s Service) PublishVersion(ctx context.Context, userId string, projectId string, versionId string) (applicationdto.VersionView, error) {
	version, err := s.loadVersionForUser(ctx, userId, projectId, versionId)
	if err != nil {
		return applicationdto.VersionView{}, err
	}
	if version.Status == status.VersionStatusPublished {
		return s.VersionForUser(ctx, userId, projectId, version.Id)
	}
	components, err := s.store.VersionComponentsByVersion(ctx, projectId, version.Id)
	if err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	if len(components) == 0 {
		return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, "At least one component is required")
	}
	if err := validateVersionComponents(components); err != nil {
		return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, err.Error())
	}
	version.Status = status.VersionStatusPublished
	if err := s.store.UpdateVersion(ctx, projectId, version); err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to publish version", err)
	}
	return s.VersionForUser(ctx, userId, projectId, version.Id)
}

func (s Service) UnpublishVersion(ctx context.Context, userId string, projectId string, versionId string) (applicationdto.VersionView, error) {
	version, err := s.loadVersionForUser(ctx, userId, projectId, versionId)
	if err != nil {
		return applicationdto.VersionView{}, err
	}
	if version.Status == status.VersionStatusUnpublished {
		return s.VersionForUser(ctx, userId, projectId, version.Id)
	}
	version.Status = status.VersionStatusUnpublished
	if err := s.store.UpdateVersion(ctx, projectId, version); err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to unpublish version", err)
	}
	return s.VersionForUser(ctx, userId, projectId, version.Id)
}

func (s Service) DeleteVersion(ctx context.Context, userId string, projectId string, versionId string) error {
	version, err := s.loadVersionForUser(ctx, userId, projectId, versionId)
	if err != nil {
		return err
	}
	services, err := s.store.ListServicesByVersion(ctx, projectId, version.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check version references", err)
	}
	if len(services) > 0 {
		return apperror.New(apperror.KindValidation, "Version is referenced and cannot be deleted")
	}
	if err := s.store.DeleteVersion(ctx, projectId, version.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete version", err)
	}
	return nil
}

func (s Service) ForkVersion(ctx context.Context, userId string, projectId string, versionId string, label string) (applicationdto.VersionView, error) {
	source, err := s.loadVersionForUser(ctx, userId, projectId, versionId)
	if err != nil {
		return applicationdto.VersionView{}, err
	}
	if strings.TrimSpace(label) == "" || len(label) > 128 {
		return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, "Invalid version label")
	}
	version, err := forkVersion(ctx, s.store, projectId, source, label, nil)
	if err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to fork version", err)
	}
	return s.VersionForUser(ctx, userId, projectId, version.Id)
}

func (s Service) versionView(ctx context.Context, projectId string, version model.Version) (applicationdto.VersionView, error) {
	components, err := s.store.VersionComponentsByVersion(ctx, projectId, version.Id)
	if err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	return applicationdto.VersionView{Version: version, Components: components}, nil
}

func (s Service) loadVersionForUser(ctx context.Context, userId string, projectId string, versionId string) (model.Version, error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return model.Version{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.Version{}, err
	}
	versionId = strings.TrimSpace(versionId)
	version, err := s.store.Version(ctx, projectId, versionId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Version{}, apperror.New(apperror.KindNotFound, "Version "+versionId+" not found")
		}
		return model.Version{}, apperror.Wrap(apperror.KindInternal, "Failed to load version", err)
	}
	return version, nil
}

func versionComponentsFromInputs(inputs []applicationdto.VersionComponentInput) ([]model.VersionComponent, error) {
	// Empty is allowed for unpublished drafts; PublishVersion enforces at least one component.
	if len(inputs) == 0 {
		return []model.VersionComponent{}, nil
	}
	components := make([]model.VersionComponent, 0, len(inputs))
	for _, input := range inputs {
		name := input.Name
		image := input.Image
		if name == "" || image == "" {
			return nil, apperror.New(apperror.KindValidation, "Component name and image are required")
		}
		if !applicationCreateCodePattern.MatchString(name) {
			return nil, apperror.New(apperror.KindValidation, "Component name must match ^[a-z][a-z0-9-]*$")
		}
		if !validImagePullPolicy(input.PullPolicy) {
			return nil, apperror.New(apperror.KindValidation, "Component pull_policy must be always, missing or never")
		}
		entrypoint, err := parseComponentCommand(input.Entrypoint)
		if err != nil {
			return nil, err
		}
		command, err := parseComponentCommand(input.Command)
		if err != nil {
			return nil, err
		}
		healthcheck, err := componentHealthcheckFromInput(input.Healthcheck)
		if err != nil {
			return nil, err
		}
		components = append(components, model.VersionComponent{
			Name: name, Image: image,
			Entrypoint: entrypoint, Command: command,
			Env: append([]model.VersionComponentEnv(nil), input.Env...), Endpoints: append([]model.VersionComponentEndpoint(nil), input.Endpoints...),
			Mounts:       append([]model.VersionComponentMount(nil), input.Mounts...),
			Dependencies: append([]model.VersionComponentDependency(nil), input.Dependencies...), Healthcheck: healthcheck,
			Resources: cloneComponentResources(input.Resources), PullPolicy: input.PullPolicy, RestartPolicy: input.RestartPolicy,
			Tmpfs: append([]model.VersionComponentTmpfs(nil), input.Tmpfs...), Ulimits: append([]model.VersionComponentUlimit(nil), input.Ulimits...),
			Devices: cloneComponentDeviceRequests(input.Devices),
		})
	}
	return components, nil
}

func cloneVersionDefinitionComponents(input []model.VersionComponent) []model.VersionComponent {
	components := make([]model.VersionComponent, 0, len(input))
	for _, component := range input {
		copy := component
		copy.Id = ""
		copy.VersionId = ""
		copy.Entrypoint = append([]string(nil), component.Entrypoint...)
		copy.Command = append([]string(nil), component.Command...)
		copy.Env = append([]model.VersionComponentEnv(nil), component.Env...)
		copy.Endpoints = append([]model.VersionComponentEndpoint(nil), component.Endpoints...)
		copy.Mounts = append([]model.VersionComponentMount(nil), component.Mounts...)
		copy.Dependencies = append([]model.VersionComponentDependency(nil), component.Dependencies...)
		copy.Tmpfs = append([]model.VersionComponentTmpfs(nil), component.Tmpfs...)
		copy.Ulimits = append([]model.VersionComponentUlimit(nil), component.Ulimits...)
		copy.Devices = cloneComponentDeviceRequests(component.Devices)
		copy.Resources = cloneComponentResources(component.Resources)
		components = append(components, copy)
	}
	return components
}

func parseComponentCommand(input string) ([]string, error) {
	command, err := commandline.Parse(input)
	if err != nil {
		return nil, apperror.New(apperror.KindValidation, "Invalid component command")
	}
	return command, nil
}

func componentHealthcheckFromInput(input *applicationdto.VersionComponentHealthcheckInput) (*model.VersionComponentHealthcheck, error) {
	if input == nil {
		return nil, nil
	}
	return &model.VersionComponentHealthcheck{
		TestMode:      input.TestMode,
		Test:          input.Test,
		Interval:      input.Interval,
		Timeout:       input.Timeout,
		Retries:       input.Retries,
		StartPeriod:   input.StartPeriod,
		StartInterval: input.StartInterval,
		Disabled:      input.Disabled,
	}, nil
}

func cloneComponentResources(input *model.VersionComponentResources) *model.VersionComponentResources {
	if input == nil {
		return nil
	}
	copy := *input
	return &copy
}

func cloneComponentDeviceRequests(input []model.VersionComponentDeviceRequest) []model.VersionComponentDeviceRequest {
	devices := make([]model.VersionComponentDeviceRequest, 0, len(input))
	for _, device := range input {
		devices = append(devices, model.VersionComponentDeviceRequest{
			Driver: device.Driver, Count: device.Count, Capabilities: append([]string(nil), device.Capabilities...),
		})
	}
	return devices
}
