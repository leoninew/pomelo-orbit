package applicationsvc

import (
	"context"
	"errors"
	"strconv"
	"strings"

	applicationdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/common/commandline"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func (s Service) ListVersions(ctx context.Context, userId string, applicationId string) ([]applicationdto.VersionView, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return nil, err
	}
	versions, err := s.store.ListVersions(ctx, app.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list versions", err)
	}
	views := make([]applicationdto.VersionView, 0, len(versions))
	for _, version := range versions {
		view, err := s.versionView(ctx, version)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func (s Service) ListVersionsPage(ctx context.Context, userId string, applicationId string, page int, perPage int, search string) (repository.Page[applicationdto.VersionView], error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return repository.Page[applicationdto.VersionView]{}, err
	}
	versions, err := s.store.ListVersionsPage(ctx, app.Id, page, perPage, search)
	if err != nil {
		return repository.Page[applicationdto.VersionView]{}, apperror.Wrap(apperror.KindInternal, "Failed to list versions", err)
	}
	views := make([]applicationdto.VersionView, 0, len(versions.Items))
	for _, version := range versions.Items {
		views = append(views, applicationdto.VersionView{Version: version})
	}
	return repository.Page[applicationdto.VersionView]{Items: views, Total: versions.Total, Page: versions.Page, PerPage: versions.PerPage}, nil
}

func (s Service) VersionForUser(ctx context.Context, userId string, versionId string) (applicationdto.VersionView, error) {
	version, err := s.loadVersionForUser(ctx, userId, versionId)
	if err != nil {
		return applicationdto.VersionView{}, err
	}
	return s.versionView(ctx, version)
}

func (s Service) CreateVersion(ctx context.Context, userId string, input applicationdto.VersionCreateInput) (applicationdto.VersionView, error) {
	app, err := s.loadApplicationForUser(ctx, userId, input.ApplicationId)
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
	exposes, err := versionExposesFromInputs(input.Exposes)
	if err != nil {
		return applicationdto.VersionView{}, err
	}
	if err := validateVersionExposes(exposes, components); err != nil {
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
	for i := range exposes {
		exposes[i].Id = idutil.NewId()
		exposes[i].VersionId = version.Id
	}
	if err := s.store.CreateVersionWithVersionComponentsAndExposes(ctx, version, components, exposes); err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to create version", err)
	}
	return s.VersionForUser(ctx, userId, version.Id)
}

func (s Service) UpdateVersion(ctx context.Context, userId string, versionId string, input applicationdto.VersionUpdateInput) (applicationdto.VersionView, error) {
	version, err := s.loadVersionForUser(ctx, userId, versionId)
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
	components, err := s.store.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	if err := s.validateServicesRuntimeConfig(ctx, version, components); err != nil {
		return applicationdto.VersionView{}, err
	}
	if err := s.store.UpdateVersion(ctx, version); err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to update version", err)
	}
	if input.Exposes != nil {
		exposes, err := versionExposesFromInputs(*input.Exposes)
		if err != nil {
			return applicationdto.VersionView{}, err
		}
		if err := validateVersionExposes(exposes, components); err != nil {
			return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, err.Error())
		}
		for i := range exposes {
			exposes[i].Id = idutil.NewId()
			exposes[i].VersionId = version.Id
		}
		if err := s.store.ReplaceVersionExposes(ctx, version.Id, exposes); err != nil {
			return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to replace exposes", err)
		}
	}
	return s.VersionForUser(ctx, userId, version.Id)
}

func (s Service) VersionComponentForUser(ctx context.Context, userId string, versionId string, componentId string) (model.VersionComponent, error) {
	version, err := s.loadVersionForUser(ctx, userId, versionId)
	if err != nil {
		return model.VersionComponent{}, err
	}
	component, err := s.store.VersionComponent(ctx, strings.TrimSpace(componentId))
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

func (s Service) CreateVersionComponent(ctx context.Context, userId string, versionId string, input applicationdto.VersionComponentInput) (model.VersionComponent, error) {
	version, err := s.loadVersionForUser(ctx, userId, versionId)
	if err != nil {
		return model.VersionComponent{}, err
	}
	if version.Status != status.VersionStatusUnpublished {
		return model.VersionComponent{}, apperror.New(apperror.KindValidation, "Published version components cannot be changed")
	}
	components, err := s.store.VersionComponentsByVersion(ctx, version.Id)
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
	if err := s.validateServicesRuntimeConfig(ctx, version, components); err != nil {
		return model.VersionComponent{}, err
	}
	if err := s.store.CreateVersionComponent(ctx, component); err != nil {
		return model.VersionComponent{}, apperror.Wrap(apperror.KindInternal, "Failed to create component", err)
	}
	return s.VersionComponentForUser(ctx, userId, version.Id, component.Id)
}

func (s Service) UpdateVersionComponentBasic(ctx context.Context, userId string, versionId string, componentId string, input applicationdto.VersionComponentBasicUpdateInput) (model.VersionComponent, error) {
	command, err := parseComponentCommand(input.Command)
	if err != nil {
		return model.VersionComponent{}, err
	}
	return s.updateVersionComponentGroup(ctx, userId, versionId, componentId, func(component *model.VersionComponent) {
		component.Name = input.Name
		component.Image = input.Image
		component.Command = command
		component.PullPolicy = input.PullPolicy
		component.RestartPolicy = input.RestartPolicy
	}, s.store.UpdateVersionComponentBasic)
}

func (s Service) UpdateVersionComponentRuntime(ctx context.Context, userId string, versionId string, componentId string, input applicationdto.VersionComponentRuntimeUpdateInput) (model.VersionComponent, error) {
	healthcheck, err := componentHealthcheckFromInput(input.Healthcheck)
	if err != nil {
		return model.VersionComponent{}, err
	}
	return s.updateVersionComponentGroup(ctx, userId, versionId, componentId, func(component *model.VersionComponent) {
		component.Healthcheck = healthcheck
	}, func(ctx context.Context, component model.VersionComponent, _ string) error {
		return s.store.UpdateVersionComponentRuntime(ctx, component)
	})
}

func (s Service) UpdateVersionComponentPorts(ctx context.Context, userId string, versionId string, componentId string, input applicationdto.VersionComponentPortsUpdateInput) (model.VersionComponent, error) {
	return s.updateVersionComponentGroup(ctx, userId, versionId, componentId, func(component *model.VersionComponent) {
		component.Ports = append([]model.VersionComponentPort(nil), input.Ports...)
	}, func(ctx context.Context, component model.VersionComponent, _ string) error {
		return s.store.UpdateVersionComponentPorts(ctx, component)
	})
}

func (s Service) UpdateVersionComponentEnv(ctx context.Context, userId string, versionId string, componentId string, input applicationdto.VersionComponentEnvUpdateInput) (model.VersionComponent, error) {
	return s.updateVersionComponentGroup(ctx, userId, versionId, componentId, func(component *model.VersionComponent) {
		component.Env = append([]model.VersionComponentEnv(nil), input.Env...)
	}, func(ctx context.Context, component model.VersionComponent, _ string) error {
		return s.store.UpdateVersionComponentEnv(ctx, component)
	})
}

func (s Service) UpdateVersionComponentMounts(ctx context.Context, userId string, versionId string, componentId string, input applicationdto.VersionComponentMountsUpdateInput) (model.VersionComponent, error) {
	return s.updateVersionComponentGroup(ctx, userId, versionId, componentId, func(component *model.VersionComponent) {
		component.Mounts = append([]model.VersionComponentMount(nil), input.Mounts...)
	}, func(ctx context.Context, component model.VersionComponent, _ string) error {
		return s.store.UpdateVersionComponentMounts(ctx, component)
	})
}

func (s Service) UpdateVersionComponentDependencies(ctx context.Context, userId string, versionId string, componentId string, input applicationdto.VersionComponentDependenciesUpdateInput) (model.VersionComponent, error) {
	return s.updateVersionComponentGroup(ctx, userId, versionId, componentId, func(component *model.VersionComponent) {
		component.Dependencies = append([]model.VersionComponentDependency(nil), input.Dependencies...)
	}, func(ctx context.Context, component model.VersionComponent, _ string) error {
		return s.store.UpdateVersionComponentDependencies(ctx, component)
	})
}

func (s Service) UpdateVersionComponentAdvanced(ctx context.Context, userId string, versionId string, componentId string, input applicationdto.VersionComponentAdvancedUpdateInput) (model.VersionComponent, error) {
	return s.updateVersionComponentGroup(ctx, userId, versionId, componentId, func(component *model.VersionComponent) {
		component.Resources = cloneComponentResources(input.Resources)
		component.Tmpfs = append([]model.VersionComponentTmpfs(nil), input.Tmpfs...)
		component.Ulimits = append([]model.VersionComponentUlimit(nil), input.Ulimits...)
	}, func(ctx context.Context, component model.VersionComponent, _ string) error {
		return s.store.UpdateVersionComponentAdvanced(ctx, component)
	})
}

func (s Service) updateVersionComponentGroup(ctx context.Context, userId string, versionId string, componentId string, update func(*model.VersionComponent), persist func(context.Context, model.VersionComponent, string) error) (model.VersionComponent, error) {
	version, err := s.loadVersionForUser(ctx, userId, versionId)
	if err != nil {
		return model.VersionComponent{}, err
	}
	if version.Status != status.VersionStatusUnpublished {
		return model.VersionComponent{}, apperror.New(apperror.KindValidation, "Published version components cannot be changed")
	}
	existing, err := s.VersionComponentForUser(ctx, userId, version.Id, componentId)
	if err != nil {
		return model.VersionComponent{}, err
	}
	components, err := s.store.VersionComponentsByVersion(ctx, version.Id)
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
	if err := s.validateServicesRuntimeConfig(ctx, version, components); err != nil {
		return model.VersionComponent{}, err
	}
	if err := persist(ctx, component, existing.Name); err != nil {
		return model.VersionComponent{}, apperror.Wrap(apperror.KindInternal, "Failed to update component", err)
	}
	return s.VersionComponentForUser(ctx, userId, version.Id, component.Id)
}

func (s Service) DeleteVersionComponent(ctx context.Context, userId string, versionId string, componentId string) error {
	version, err := s.loadVersionForUser(ctx, userId, versionId)
	if err != nil {
		return err
	}
	if version.Status != status.VersionStatusUnpublished {
		return apperror.New(apperror.KindValidation, "Published version components cannot be changed")
	}
	component, err := s.VersionComponentForUser(ctx, userId, version.Id, componentId)
	if err != nil {
		return err
	}
	components, err := s.store.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	exposes, err := s.store.VersionExposesByVersion(ctx, version.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to list exposes", err)
	}
	references := make([]string, 0)
	for _, expose := range exposes {
		if expose.ComponentName == component.Name {
			references = append(references, "expose "+expose.Protocol+":"+strconv.Itoa(expose.ContainerPort))
		}
	}
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
	if err := s.store.DeleteVersionComponent(ctx, component); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete component", err)
	}
	return nil
}

func (s Service) PublishVersion(ctx context.Context, userId string, versionId string) (applicationdto.VersionView, error) {
	version, err := s.loadVersionForUser(ctx, userId, versionId)
	if err != nil {
		return applicationdto.VersionView{}, err
	}
	if version.Status == status.VersionStatusPublished {
		return s.VersionForUser(ctx, userId, version.Id)
	}
	components, err := s.store.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	if len(components) == 0 {
		return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, "At least one component is required")
	}
	if err := validateVersionComponents(components); err != nil {
		return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, err.Error())
	}
	if err := s.validateServicesRuntimeConfig(ctx, version, components); err != nil {
		return applicationdto.VersionView{}, err
	}
	exposes, err := s.store.VersionExposesByVersion(ctx, version.Id)
	if err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to list exposes", err)
	}
	if err := validateVersionExposes(exposes, components); err != nil {
		return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, err.Error())
	}
	version.Status = status.VersionStatusPublished
	if err := s.store.UpdateVersion(ctx, version); err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to publish version", err)
	}
	return s.VersionForUser(ctx, userId, version.Id)
}

func (s Service) UnpublishVersion(ctx context.Context, userId string, versionId string) (applicationdto.VersionView, error) {
	version, err := s.loadVersionForUser(ctx, userId, versionId)
	if err != nil {
		return applicationdto.VersionView{}, err
	}
	if version.Status == status.VersionStatusUnpublished {
		return s.VersionForUser(ctx, userId, version.Id)
	}
	version.Status = status.VersionStatusUnpublished
	if err := s.store.UpdateVersion(ctx, version); err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to unpublish version", err)
	}
	return s.VersionForUser(ctx, userId, version.Id)
}

func (s Service) DeleteVersion(ctx context.Context, userId string, versionId string) error {
	version, err := s.loadVersionForUser(ctx, userId, versionId)
	if err != nil {
		return err
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

func (s Service) ForkVersion(ctx context.Context, userId string, versionId string, label string) (applicationdto.VersionView, error) {
	source, err := s.loadVersionForUser(ctx, userId, versionId)
	if err != nil {
		return applicationdto.VersionView{}, err
	}
	if strings.TrimSpace(label) == "" || len(label) > 128 {
		return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, "Invalid version label")
	}
	components, err := s.store.VersionComponentsByVersion(ctx, source.Id)
	if err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	exposes, err := s.store.VersionExposesByVersion(ctx, source.Id)
	if err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to list exposes", err)
	}
	fromId := source.Id
	version := model.Version{
		Id:                   idutil.NewId(),
		ApplicationId:        source.ApplicationId,
		Label:                label,
		Status:               status.VersionStatusUnpublished,
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
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to fork version", err)
	}
	return s.VersionForUser(ctx, userId, version.Id)
}

func (s Service) versionView(ctx context.Context, version model.Version) (applicationdto.VersionView, error) {
	components, err := s.store.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	exposes, err := s.store.VersionExposesByVersion(ctx, version.Id)
	if err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to list exposes", err)
	}
	return applicationdto.VersionView{Version: version, Components: components, Exposes: exposes}, nil
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
		pullPolicy := "missing"
		if input.PullPolicy != nil {
			pullPolicy = *input.PullPolicy
		}
		if !validImagePullPolicy(pullPolicy) {
			return nil, apperror.New(apperror.KindValidation, "Component pull_policy must be always, missing or never")
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
			Command: command,
			Env:     append([]model.VersionComponentEnv(nil), input.Env...), Ports: append([]model.VersionComponentPort(nil), input.Ports...),
			Mounts:       append([]model.VersionComponentMount(nil), input.Mounts...),
			Dependencies: append([]model.VersionComponentDependency(nil), input.Dependencies...), Healthcheck: healthcheck,
			Resources: cloneComponentResources(input.Resources), PullPolicy: &pullPolicy, RestartPolicy: input.RestartPolicy,
			Tmpfs: append([]model.VersionComponentTmpfs(nil), input.Tmpfs...), Ulimits: append([]model.VersionComponentUlimit(nil), input.Ulimits...),
		})
	}
	return components, nil
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

func cloneComponentHealthcheck(input *model.VersionComponentHealthcheck) *model.VersionComponentHealthcheck {
	if input == nil {
		return nil
	}
	copy := *input
	return &copy
}

func cloneComponentResources(input *model.VersionComponentResources) *model.VersionComponentResources {
	if input == nil {
		return nil
	}
	copy := *input
	return &copy
}

func versionExposesFromInputs(inputs []applicationdto.VersionExposeInput) ([]model.VersionExpose, error) {
	exposes := make([]model.VersionExpose, 0, len(inputs))
	for _, input := range inputs {
		protocol := input.Protocol
		componentName := input.ComponentName
		if componentName == "" || (protocol != "http" && protocol != "tcp") || input.ContainerPort < 1 || input.ContainerPort > 65535 {
			return nil, apperror.New(apperror.KindValidation, "Invalid expose fields")
		}
		access := input.Access
		if access != exposeAccessLocal && access != exposeAccessPublic {
			return nil, apperror.New(apperror.KindValidation, "expose access must be local or public")
		}
		var listenPort *int
		if input.ListenPort != nil {
			if *input.ListenPort < 1 || *input.ListenPort > 65535 {
				return nil, apperror.New(apperror.KindValidation, "expose listen_port out of range")
			}
			v := *input.ListenPort
			listenPort = &v
		}
		pathPrefix := optionalText(input.PathPrefix)
		if protocol == "tcp" && pathPrefix != nil && *pathPrefix != "" {
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
