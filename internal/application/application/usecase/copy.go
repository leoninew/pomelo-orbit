package applicationsvc

import (
	"context"
	"errors"
	"time"

	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func (s Service) CopyApplication(ctx context.Context, userId, sourceProjectId, targetProjectId, applicationId string, copyAllVersions bool) (applicationdto.ApplicationCopyResult, error) {
	definition, err := s.ApplicationDefinitionForUser(ctx, userId, sourceProjectId, applicationId)
	if err != nil {
		return applicationdto.ApplicationCopyResult{}, err
	}
	if err := s.ensureProjectMembership(ctx, targetProjectId, userId); err != nil {
		return applicationdto.ApplicationCopyResult{}, err
	}
	if err := s.ensureCopyNameAvailable(ctx, targetProjectId, definition.Application.Name); err != nil {
		return applicationdto.ApplicationCopyResult{}, err
	}
	if err := s.ensureCopyCodeAvailable(ctx, targetProjectId, definition.Application.Code); err != nil {
		return applicationdto.ApplicationCopyResult{}, err
	}

	if !copyAllVersions {
		definition.Versions = definition.Versions[:1]
	}

	projectId := targetProjectId
	application := definition.Application
	application.Id = idutil.NewId()
	application.ProjectId = &projectId
	application.CreatedAt = time.Time{}
	application.UpdatedAt = time.Time{}
	if err := s.store.CreateApplication(ctx, application); err != nil {
		return applicationdto.ApplicationCopyResult{}, apperror.Wrap(apperror.KindInternal, "Failed to copy application", err)
	}

	versionIds := make(map[string]string, len(definition.Versions))
	componentIds := make(map[string]string)
	versionsBySourceId := make(map[string]applicationdto.VersionDefinition, len(definition.Versions))
	// Version lists use descending IDs, so allocate new IDs oldest first.
	for index := len(definition.Versions) - 1; index >= 0; index-- {
		version := definition.Versions[index]
		versionIds[version.Version.Id] = idutil.NewId()
		versionsBySourceId[version.Version.Id] = version
		for _, component := range version.Components {
			componentIds[component.Id] = idutil.NewId()
		}
	}

	created := make(map[string]bool, len(definition.Versions))
	var createVersion func(string) error
	createVersion = func(sourceId string) error {
		if created[sourceId] {
			return nil
		}
		definition := versionsBySourceId[sourceId]
		version := definition.Version
		version.Id = versionIds[sourceId]
		version.ApplicationId = application.Id
		version.CreatedAt = time.Time{}
		version.UpdatedAt = time.Time{}
		if !copyAllVersions {
			version.CreatedFromVersionId = nil
		} else if version.CreatedFromVersionId != nil {
			parentSourceId := *version.CreatedFromVersionId
			if err := createVersion(parentSourceId); err != nil {
				return err
			}
			parentId := versionIds[parentSourceId]
			version.CreatedFromVersionId = &parentId
		}

		components := make([]model.VersionComponent, len(definition.Components))
		for index, sourceComponent := range definition.Components {
			component := sourceComponent
			component.Id = componentIds[sourceComponent.Id]
			component.VersionId = version.Id
			component.CreatedAt = time.Time{}
			component.UpdatedAt = time.Time{}
			components[index] = component
		}
		if err := s.store.CreateVersionWithVersionComponents(ctx, targetProjectId, version, components); err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to copy version", err)
		}
		created[sourceId] = true
		return nil
	}
	for _, version := range definition.Versions {
		if err := createVersion(version.Version.Id); err != nil {
			return applicationdto.ApplicationCopyResult{}, err
		}
	}

	return applicationdto.ApplicationCopyResult{
		Application: application, VersionIds: versionIds, VersionComponentIds: componentIds,
		LatestVersionId: versionIds[definition.Versions[0].Version.Id],
	}, nil
}

func (s Service) ensureCopyNameAvailable(ctx context.Context, projectId, name string) error {
	if _, err := s.store.ApplicationByProjectAndName(ctx, projectId, name); err == nil {
		return apperror.New(apperror.KindConflict, "Application name already exists in the target project")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check target application name", err)
	}
	return nil
}

func (s Service) ensureCopyCodeAvailable(ctx context.Context, projectId, code string) error {
	if _, err := s.store.ApplicationByProjectAndCode(ctx, projectId, code); err == nil {
		return apperror.New(apperror.KindConflict, "Application code already exists in the target project")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check target application code", err)
	}
	return nil
}
