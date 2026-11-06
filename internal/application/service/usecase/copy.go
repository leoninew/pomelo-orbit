package servicesvc

import (
	"context"
	"errors"
	"time"

	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/common/workspacepath"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type applicationCopier interface {
	CopyApplication(context.Context, string, string, string, string, bool) (applicationdto.ApplicationCopyResult, error)
}

type copyEnvironmentReader interface {
	EnvironmentForUser(context.Context, string, string) (environmentdto.View, error)
}

type CopyService struct {
	service     Service
	application applicationCopier
	environment copyEnvironmentReader
}

func NewCopyService(service Service, application applicationCopier, environment copyEnvironmentReader) CopyService {
	return CopyService{service: service, application: application, environment: environment}
}

func (s CopyService) Copy(ctx context.Context, userId, sourceProjectId, serviceId string, input servicedto.ServiceCopyInput) (model.Service, error) {
	targetProjectId, copyAllVersions := input.TargetProjectId, input.CopyAllVersions
	if sourceProjectId == "" || targetProjectId == "" || serviceId == "" {
		return model.Service{}, apperror.New(apperror.KindValidation, "source project, target project and service are required")
	}
	if sourceProjectId == targetProjectId {
		return model.Service{}, apperror.New(apperror.KindValidation, "target project must differ from source project")
	}

	definition, err := s.service.ServiceDefinitionForUser(ctx, userId, sourceProjectId, serviceId)
	if err != nil {
		return model.Service{}, err
	}
	environment, err := s.environment.EnvironmentForUser(ctx, userId, targetProjectId)
	if err != nil {
		return model.Service{}, err
	}
	if input.EnvironmentTargetRevision != environment.TargetRevision {
		return model.Service{}, apperror.New(apperror.KindConflict, "Environment changed. Confirm the deployment directory again.")
	}
	var platform string
	if environment.TargetType == model.EnvironmentTargetTypeSSH {
		platform = environment.SSH.Platform
	} else {
		platform = environment.Local.Platform
	}
	directory, err := workspacepath.NormalizeServiceDirectory(input.DeploymentDirectory, platform)
	if err != nil {
		return model.Service{}, apperror.New(apperror.KindValidation, err.Error())
	}

	application, err := s.application.CopyApplication(ctx, userId, sourceProjectId, targetProjectId, definition.Service.ApplicationId, copyAllVersions)
	if err != nil {
		return model.Service{}, err
	}

	targetVersionId := application.VersionIds[definition.Service.VersionId]
	if !copyAllVersions {
		targetVersionId = application.LatestVersionId
	}
	targetDeclarations, err := s.service.application.VersionComponentsByVersion(ctx, targetProjectId, targetVersionId)
	if err != nil {
		return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to load copied version components", err)
	}

	componentIds := application.VersionComponentIds
	if !copyAllVersions {
		if !sameServiceComponentNames(definition.Components, targetDeclarations) {
			return model.Service{}, apperror.New(apperror.KindConflict, "Service components differ from the latest version; choose all versions to preserve the service configuration")
		}
		componentIdsByName := make(map[string]string, len(targetDeclarations))
		for _, declaration := range targetDeclarations {
			componentIdsByName[declaration.Name] = declaration.Id
		}
		componentIds = make(map[string]string, len(definition.Components))
		for _, component := range definition.Components {
			componentIds[component.SourceVersionComponentId] = componentIdsByName[component.ComponentName]
		}
	}

	if _, err := s.service.service.ServiceByProjectAndCode(ctx, targetProjectId, definition.Service.Code); err == nil {
		return model.Service{}, apperror.New(apperror.KindConflict, "Service code already exists in the target project")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to check target service code", err)
	}

	service := definition.Service
	service.Id = idutil.NewId()
	service.ProjectId = targetProjectId
	service.ApplicationId = application.Application.Id
	service.VersionId = targetVersionId
	service.CurrentDeploymentId = nil
	service.RuntimeDirectory = ""
	service.DeploymentDirectory = directory
	service.DirectoryTargetRevision = environment.TargetRevision
	service.RuntimeTargetRevision = 0
	service.Status = status.ServiceStatusStopped
	service.CreatedAt = time.Time{}
	service.UpdatedAt = time.Time{}

	components := make([]model.ServiceComponent, len(definition.Components))
	for index, sourceComponent := range definition.Components {
		component := sourceComponent
		component.Id = idutil.NewId()
		component.ServiceId = service.Id
		component.SourceVersionComponentId = componentIds[sourceComponent.SourceVersionComponentId]
		component.CreatedAt = time.Time{}
		component.UpdatedAt = time.Time{}
		components[index] = component
	}
	if err := s.service.service.CreateServiceWithComponents(ctx, targetProjectId, service, components); err != nil {
		return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to create copied service", err)
	}
	if len(definition.Env) > 0 {
		if err := s.service.service.ReplaceServiceEnv(ctx, targetProjectId, service.Id, definition.Env); err != nil {
			return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to save copied service environment", err)
		}
	}
	if err := s.service.service.UpdateServiceDeploymentDirectory(ctx, targetProjectId, service.Id, service.DeploymentDirectory, service.DirectoryTargetRevision); err != nil {
		return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to save copied service deployment directory", err)
	}
	return service, nil
}

func sameServiceComponentNames(components []model.ServiceComponent, declarations []model.VersionComponent) bool {
	if len(components) != len(declarations) {
		return false
	}
	names := make(map[string]struct{}, len(declarations))
	for _, declaration := range declarations {
		names[declaration.Name] = struct{}{}
	}
	for _, component := range components {
		if _, ok := names[component.ComponentName]; !ok {
			return false
		}
	}
	return true
}
