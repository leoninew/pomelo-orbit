package applicationsvc

import (
	"context"
	"errors"
	"strings"

	applicationdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func (s Service) ListVersions(ctx context.Context, userId string, applicationId string) ([]applicationdto.VersionView, error) {
	page, err := s.ListVersionsPage(ctx, userId, applicationId, 1, 100, "")
	if err != nil {
		return nil, err
	}
	return page.Items, nil
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
		view, err := s.versionView(ctx, version)
		if err != nil {
			return repository.Page[applicationdto.VersionView]{}, err
		}
		views = append(views, view)
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
	label := strings.TrimSpace(input.Label)
	if label == "" || len(label) > 128 {
		return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, "Invalid version label")
	}
	components, err := normalizeVersionComponents(input.Components)
	if err != nil {
		return applicationdto.VersionView{}, err
	}
	if err := validateVersionComponents(components); err != nil {
		return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, err.Error())
	}
	if err := s.validateRuntimeEnvReferences(ctx, app, components); err != nil {
		return applicationdto.VersionView{}, err
	}
	exposes, err := normalizeVersionExposes(input.Exposes)
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
		EnvJSON:       normalizeOptionalText(input.EnvJSON),
		Note:          normalizeOptionalText(input.Note),
	}
	for i := range components {
		components[i].Id = idutil.NewId()
		components[i].VersionId = version.Id
		for j := range components[i].SecretEnvRefs {
			components[i].SecretEnvRefs[j].ComponentId = components[i].Id
		}
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
		label := strings.TrimSpace(*input.Label)
		if label == "" || len(label) > 128 {
			return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, "Invalid version label")
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
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to update version", err)
	}
	var components []model.VersionComponent
	if input.Components != nil {
		components, err = normalizeVersionComponents(*input.Components)
		if err != nil {
			return applicationdto.VersionView{}, err
		}
		if err := validateVersionComponents(components); err != nil {
			return applicationdto.VersionView{}, apperror.New(apperror.KindValidation, err.Error())
		}
		app, err := s.store.Application(ctx, version.ApplicationId)
		if err != nil {
			return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
		}
		if err := s.validateRuntimeEnvReferences(ctx, app, components); err != nil {
			return applicationdto.VersionView{}, err
		}
		for i := range components {
			components[i].Id = idutil.NewId()
			components[i].VersionId = version.Id
			for j := range components[i].SecretEnvRefs {
				components[i].SecretEnvRefs[j].ComponentId = components[i].Id
			}
		}
		if err := s.store.ReplaceVersionComponents(ctx, version.Id, components); err != nil {
			return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to replace components", err)
		}
	} else {
		components, err = s.store.VersionComponentsByVersion(ctx, version.Id)
		if err != nil {
			return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
		}
	}
	if input.Exposes != nil {
		exposes, err := normalizeVersionExposes(*input.Exposes)
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
	app, err := s.store.Application(ctx, version.ApplicationId)
	if err != nil {
		return applicationdto.VersionView{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	if err := s.validateRuntimeEnvReferences(ctx, app, components); err != nil {
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
	label = strings.TrimSpace(label)
	if label == "" || len(label) > 128 {
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
		EnvJSON:              source.EnvJSON,
		CreatedFromVersionId: &fromId,
		Note:                 source.Note,
	}
	forkedComponents := make([]model.VersionComponent, 0, len(components))
	for _, component := range components {
		component.Id = idutil.NewId()
		component.VersionId = version.Id
		for i := range component.SecretEnvRefs {
			component.SecretEnvRefs[i].ComponentId = component.Id
		}
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

func normalizeVersionComponents(inputs []applicationdto.VersionComponentInput) ([]model.VersionComponent, error) {
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
			RestartPolicy:   input.RestartPolicy,
			TmpfsJSON:       input.TmpfsJSON,
			UlimitsJSON:     input.UlimitsJSON,
			SecretEnvRefs:   secretEnvRefsFromInput(input.SecretEnvRefs),
		})
	}
	return components, nil
}

func normalizeVersionExposes(inputs []applicationdto.VersionExposeInput) ([]model.VersionExpose, error) {
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

func secretEnvRefsFromInput(inputs []applicationdto.VersionComponentSecretEnvRefInput) []model.VersionComponentSecretEnvRef {
	refs := make([]model.VersionComponentSecretEnvRef, 0, len(inputs))
	for _, input := range inputs {
		refs = append(refs, model.VersionComponentSecretEnvRef{
			EnvKey:       input.EnvKey,
			CredentialId: input.CredentialId,
			DataKey:      input.DataKey,
		})
	}
	return refs
}
