package handoversvc

import (
	"context"
	"strings"

	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	projectdto "github.com/leoninew/pomelo-orbit/internal/application/project/dto"
	handoverdto "github.com/leoninew/pomelo-orbit/internal/application/project_handover/dto"
	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

const handoverPageSize = 10_000

type projectDomain interface {
	LoadForUser(context.Context, string, string) (model.Project, error)
	CreateFromDefinition(context.Context, string, projectdto.ProjectDefinition) (model.Project, error)
}

type environmentDomain interface {
	TargetDefinitionForUser(context.Context, string, string) (environmentdto.TargetDefinition, error)
	SaveTargetDefinitionForHandover(context.Context, string, string, environmentdto.TargetDefinition) (environmentdto.TargetDefinition, error)
}

type applicationDomain interface {
	ListApplications(context.Context, string, string, int, int, string, string) (repository.Page[model.Application], error)
	ApplicationDefinitionForUser(context.Context, string, string, string) (applicationdto.ApplicationDefinition, error)
	CreateApplicationFromDefinition(context.Context, string, string, applicationdto.ApplicationDefinition) (applicationdto.ApplicationDefinition, error)
	RemoveApplication(context.Context, string, string, string) error
}

type serviceDomain interface {
	ListServices(context.Context, string, servicedto.ServiceListInput) (repository.Page[servicedto.ServiceView], error)
	ServiceDefinitionForUser(context.Context, string, string, string) (servicedto.ServiceDefinition, error)
	CreateServiceFromDefinition(context.Context, string, string, servicedto.ServiceDefinition) (servicedto.ServiceView, error)
	RemoveService(context.Context, string, string, string) error
}

type routeDomain interface {
	ListAllRoutes(context.Context, string, string) ([]model.Route, error)
	CreateRouteFromDefinition(context.Context, string, string, routedto.RouteDefinitionInput) (model.Route, error)
	RemoveRoute(context.Context, string, string, string) error
}

type gatewayDomain interface {
	ListGateways(context.Context, string, string, int, int, string) (repository.Page[gatewaydto.GatewayView], error)
	GatewayDefinitionForUser(context.Context, string, string, string) (gatewaydto.GatewayDefinition, error)
	CreateGatewayFromDefinition(context.Context, string, string, gatewaydto.GatewayDefinition) (gatewaydto.GatewayDefinition, error)
	RemoveGateway(context.Context, string, string, string) error
}

type pipelineDomain interface {
	EnsureNoCDConfigurationReferences(context.Context, string, string) error
}

type pipelineRunDomain interface {
	EnsureNoCDConfigurationReferences(context.Context, string, string) error
}

type deploymentDomain interface {
	ClearProjectDeploymentHistory(context.Context, string, string) error
}

// Service is an application-layer coordinator. All persistent reads and
// writes go through the business domains; it owns only package format and
// source-to-target identity mapping.
type Service struct {
	project     projectDomain
	environment environmentDomain
	application applicationDomain
	service     serviceDomain
	route       routeDomain
	gateway     gatewayDomain
	pipeline    pipelineDomain
	pipelineRun pipelineRunDomain
	deployment  deploymentDomain
	secretKey   string
}

func New(project projectDomain, environment environmentDomain, application applicationDomain, service serviceDomain, route routeDomain, gateway gatewayDomain, pipeline pipelineDomain, pipelineRun pipelineRunDomain, deployment deploymentDomain, secretKey string) Service {
	return Service{project: project, environment: environment, application: application, service: service, route: route, gateway: gateway, pipeline: pipeline, pipelineRun: pipelineRun, deployment: deployment, secretKey: secretKey}
}

func (s Service) ExportDocument(ctx context.Context, userId, projectId string) ([]byte, error) {
	item, err := s.Export(ctx, userId, projectId)
	if err != nil {
		return nil, err
	}
	if err := encryptPackagePrivateKey(s.secretKey, &item); err != nil {
		return nil, err
	}
	document, err := handoverdto.Encode(item)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to encode handover package", err)
	}
	return document, nil
}

func (s Service) ImportDocument(ctx context.Context, userId string, input handoverdto.ImportInput, document []byte) (model.Project, error) {
	item, err := handoverdto.Decode(document)
	if err != nil {
		return model.Project{}, apperror.Wrap(apperror.KindValidation, handoverdto.DecodeErrorMessage(err), err)
	}
	if input.OverrideEnvironment {
		if err := decryptPackagePrivateKey(s.secretKey, input.DecryptionKey, &item); err != nil {
			return model.Project{}, err
		}
	}
	input.Package = item
	return s.Import(ctx, userId, input)
}

func encryptPackagePrivateKey(secretKey string, item *handoverdto.Package) error {
	if item.Environment.Credential == nil || item.Environment.Credential.PrivateKey == "" {
		return nil
	}
	encryptedPrivateKey, err := security.EncryptString(secretKey, item.Environment.Credential.PrivateKey)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to encrypt handover environment SSH private key", err)
	}
	item.Environment.Credential.EncryptedPrivateKey = encryptedPrivateKey
	item.Environment.Credential.PrivateKey = ""
	return nil
}

func decryptPackagePrivateKey(systemKey string, suppliedKey string, item *handoverdto.Package) error {
	if item.Environment.Credential == nil || item.Environment.Credential.EncryptedPrivateKey == "" {
		return nil
	}
	decryptionKey := strings.TrimSpace(suppliedKey)
	if decryptionKey == "" {
		decryptionKey = systemKey
	}
	privateKey, err := security.DecryptString(decryptionKey, item.Environment.Credential.EncryptedPrivateKey)
	if err != nil {
		return apperror.Wrap(apperror.KindValidation, "Failed to decrypt handover environment SSH private key", err)
	}
	item.Environment.Credential.PrivateKey = privateKey
	item.Environment.Credential.EncryptedPrivateKey = ""
	return nil
}

func (s Service) Export(ctx context.Context, userId, projectId string) (handoverdto.Package, error) {
	project, err := s.project.LoadForUser(ctx, projectId, userId)
	if err != nil {
		return handoverdto.Package{}, err
	}
	environment, err := s.environment.TargetDefinitionForUser(ctx, userId, project.Id)
	if err != nil {
		return handoverdto.Package{}, err
	}
	gateways, err := s.gateway.ListGateways(ctx, userId, project.Id, 1, handoverPageSize, "")
	if err != nil {
		return handoverdto.Package{}, err
	}
	if gateways.Total > 1 {
		return handoverdto.Package{}, apperror.New(apperror.KindValidation, "Project has multiple gateways")
	}
	var gateway *gatewaydto.GatewayDefinition
	if len(gateways.Items) == 1 {
		definition, err := s.gateway.GatewayDefinitionForUser(ctx, userId, project.Id, gateways.Items[0].Application.Id)
		if err != nil {
			return handoverdto.Package{}, err
		}
		gateway = &definition
	}
	applications, err := s.application.ListApplications(ctx, userId, project.Id, 1, handoverPageSize, "", "")
	if err != nil {
		return handoverdto.Package{}, err
	}
	if applications.Total > handoverPageSize {
		return handoverdto.Package{}, apperror.New(apperror.KindValidation, "Project has too many applications to export")
	}
	result := handoverdto.Package{
		Format: handoverdto.Format, Version: handoverdto.FormatVersion,
		Project:     projectdto.ProjectDefinition{Name: project.Name, Code: project.Code, IsActive: project.IsActive},
		Environment: environment, Applications: make([]applicationdto.ApplicationDefinition, 0, len(applications.Items)),
		Services: make([]servicedto.ServiceDefinition, 0), Routes: make([]routedto.RouteDefinitionInput, 0), Gateway: gateway,
	}
	for _, item := range applications.Items {
		if gateway != nil && item.Id == gateway.Application.Application.Id {
			continue
		}
		definition, err := s.application.ApplicationDefinitionForUser(ctx, userId, project.Id, item.Id)
		if err != nil {
			return handoverdto.Package{}, err
		}
		result.Applications = append(result.Applications, definition)
	}
	services, err := s.service.ListServices(ctx, userId, servicedto.ServiceListInput{ProjectId: project.Id, Page: 1, PerPage: handoverPageSize})
	if err != nil {
		return handoverdto.Package{}, err
	}
	if services.Total > handoverPageSize {
		return handoverdto.Package{}, apperror.New(apperror.KindValidation, "Project has too many services to export")
	}
	for _, item := range services.Items {
		if gateway != nil && item.Service.Id == gateway.RuntimeService.Service.Id {
			continue
		}
		definition, err := s.service.ServiceDefinitionForUser(ctx, userId, project.Id, item.Service.Id)
		if err != nil {
			return handoverdto.Package{}, err
		}
		result.Services = append(result.Services, definition)
	}
	routes, err := s.route.ListAllRoutes(ctx, userId, project.Id)
	if err != nil {
		return handoverdto.Package{}, err
	}
	if len(routes) > handoverPageSize {
		return handoverdto.Package{}, apperror.New(apperror.KindValidation, "Project has too many routes to export")
	}
	for _, item := range routes {
		result.Routes = append(result.Routes, routedto.RouteDefinitionInput{Route: item})
	}
	return result, nil
}

func (s Service) Import(ctx context.Context, userId string, input handoverdto.ImportInput) (model.Project, error) {
	if err := validatePackage(input.Package); err != nil {
		return model.Project{}, err
	}
	var project model.Project
	var err error
	switch input.Mode {
	case handoverdto.ImportModeNew:
		project, err = s.project.CreateFromDefinition(ctx, userId, projectdto.ProjectDefinition{
			Name:     input.Name,
			Code:     input.Code,
			IsActive: true,
		})
	case handoverdto.ImportModeReplace:
		project, err = s.project.LoadForUser(ctx, strings.TrimSpace(input.TargetProjectId), userId)
		if err == nil {
			err = s.ensureProjectConfigurationReplaceable(ctx, userId, project.Id)
		}
		if err == nil {
			err = s.removeProjectConfiguration(ctx, userId, project.Id)
		}
		if err == nil {
			err = s.deployment.ClearProjectDeploymentHistory(ctx, userId, project.Id)
		}
	default:
		return model.Project{}, apperror.New(apperror.KindValidation, "Invalid handover import mode")
	}
	if err != nil {
		return model.Project{}, err
	}
	if input.OverrideEnvironment {
		if _, err := s.environment.SaveTargetDefinitionForHandover(ctx, userId, project.Id, input.Package.Environment); err != nil {
			return model.Project{}, err
		}
	}
	return s.restoreProjectConfiguration(ctx, userId, project, input.Package)
}

func (s Service) ensureProjectConfigurationReplaceable(ctx context.Context, userId, projectId string) error {
	if s.pipeline == nil || s.pipelineRun == nil || s.deployment == nil {
		return apperror.New(apperror.KindInternal, "project handover dependencies are not configured")
	}
	if err := s.pipeline.EnsureNoCDConfigurationReferences(ctx, userId, projectId); err != nil {
		return err
	}
	return s.pipelineRun.EnsureNoCDConfigurationReferences(ctx, userId, projectId)
}

type packageIdMaps struct {
	applications map[string]string
	versions     map[string]string
	components   map[string]string
	services     map[string]string
}

func validatePackage(item handoverdto.Package) error {
	if item.Format != handoverdto.Format {
		return apperror.New(apperror.KindValidation, "Unsupported handover package format")
	}
	if item.Version != handoverdto.FormatVersion {
		return apperror.New(apperror.KindValidation, "Unsupported handover package version")
	}
	_, err := sourceDefinitionIds(item)
	return err
}

func sourceDefinitionIds(item handoverdto.Package) (packageIdMaps, error) {
	maps := packageIdMaps{
		applications: make(map[string]string, len(item.Applications)),
		versions:     make(map[string]string),
		components:   make(map[string]string),
		services:     make(map[string]string, len(item.Services)+1),
	}
	for _, definition := range item.Applications {
		applicationId := strings.TrimSpace(definition.Application.Id)
		if applicationId == "" {
			return packageIdMaps{}, apperror.New(apperror.KindValidation, "Application definition id is required")
		}
		if _, exists := maps.applications[applicationId]; exists {
			return packageIdMaps{}, apperror.New(apperror.KindValidation, "Application definition ids must be unique")
		}
		maps.applications[applicationId] = ""
		if len(definition.Versions) == 0 {
			return packageIdMaps{}, apperror.New(apperror.KindValidation, "Application definition requires a Version")
		}
		for _, version := range definition.Versions {
			versionId := strings.TrimSpace(version.Version.Id)
			if versionId == "" {
				return packageIdMaps{}, apperror.New(apperror.KindValidation, "Version definition id is required")
			}
			if _, exists := maps.versions[versionId]; exists {
				return packageIdMaps{}, apperror.New(apperror.KindValidation, "Version definition ids must be unique")
			}
			maps.versions[versionId] = ""
			for _, component := range version.Components {
				componentId := strings.TrimSpace(component.Id)
				if componentId == "" {
					return packageIdMaps{}, apperror.New(apperror.KindValidation, "Version Component definition id is required")
				}
				if _, exists := maps.components[componentId]; exists {
					return packageIdMaps{}, apperror.New(apperror.KindValidation, "Version Component definition ids must be unique")
				}
				maps.components[componentId] = ""
			}
		}
	}
	for _, definition := range item.Services {
		serviceId := strings.TrimSpace(definition.Service.Id)
		if serviceId == "" {
			return packageIdMaps{}, apperror.New(apperror.KindValidation, "Service definition id is required")
		}
		if _, exists := maps.services[serviceId]; exists {
			return packageIdMaps{}, apperror.New(apperror.KindValidation, "Service definition ids must be unique")
		}
		if _, exists := maps.applications[strings.TrimSpace(definition.Service.ApplicationId)]; !exists {
			return packageIdMaps{}, apperror.New(apperror.KindValidation, "Service definition Application is not in the package")
		}
		if _, exists := maps.versions[strings.TrimSpace(definition.Service.VersionId)]; !exists {
			return packageIdMaps{}, apperror.New(apperror.KindValidation, "Service definition Version is not in the package")
		}
		for _, component := range definition.Components {
			if _, exists := maps.components[strings.TrimSpace(component.SourceVersionComponentId)]; !exists {
				return packageIdMaps{}, apperror.New(apperror.KindValidation, "Service Component definition is not in the package")
			}
		}
		maps.services[serviceId] = ""
	}
	if item.Gateway != nil {
		gatewayServiceId := strings.TrimSpace(item.Gateway.RuntimeService.Service.Id)
		if gatewayServiceId != "" {
			if _, exists := maps.services[gatewayServiceId]; exists {
				return packageIdMaps{}, apperror.New(apperror.KindValidation, "Service definition ids must be unique")
			}
			maps.services[gatewayServiceId] = ""
		}
	}
	for _, definition := range item.Routes {
		if definition.Route.ServiceId == nil || strings.TrimSpace(*definition.Route.ServiceId) == "" {
			continue
		}
		if _, exists := maps.services[strings.TrimSpace(*definition.Route.ServiceId)]; !exists {
			return packageIdMaps{}, apperror.New(apperror.KindValidation, "Route Service is not in the package")
		}
	}
	return maps, nil
}

func (s Service) removeProjectConfiguration(ctx context.Context, userId, projectId string) error {
	gateways, err := s.gateway.ListGateways(ctx, userId, projectId, 1, handoverPageSize, "")
	if err != nil {
		return err
	}
	if gateways.Total > 1 {
		return apperror.New(apperror.KindValidation, "Project has multiple gateways")
	}
	if len(gateways.Items) == 1 {
		if err := s.gateway.RemoveGateway(ctx, userId, projectId, gateways.Items[0].Application.Id); err != nil {
			return err
		}
	}
	routes, err := s.route.ListAllRoutes(ctx, userId, projectId)
	if err != nil {
		return err
	}
	if len(routes) > handoverPageSize {
		return apperror.New(apperror.KindValidation, "Project has too many routes to replace")
	}
	for _, route := range routes {
		if err := s.route.RemoveRoute(ctx, userId, projectId, route.Id); err != nil {
			return err
		}
	}
	services, err := s.service.ListServices(ctx, userId, servicedto.ServiceListInput{ProjectId: projectId, Page: 1, PerPage: handoverPageSize})
	if err != nil {
		return err
	}
	if services.Total > handoverPageSize {
		return apperror.New(apperror.KindValidation, "Project has too many services to replace")
	}
	for _, service := range services.Items {
		if err := s.service.RemoveService(ctx, userId, projectId, service.Service.Id); err != nil {
			return err
		}
	}
	applications, err := s.application.ListApplications(ctx, userId, projectId, 1, handoverPageSize, "", "")
	if err != nil {
		return err
	}
	if applications.Total > handoverPageSize {
		return apperror.New(apperror.KindValidation, "Project has too many applications to replace")
	}
	for _, application := range applications.Items {
		if err := s.application.RemoveApplication(ctx, userId, projectId, application.Id); err != nil {
			return err
		}
	}
	return nil
}

func (s Service) restoreProjectConfiguration(ctx context.Context, userId string, project model.Project, item handoverdto.Package) (model.Project, error) {
	maps, err := sourceDefinitionIds(item)
	if err != nil {
		return model.Project{}, err
	}
	for _, definition := range item.Applications {
		created, err := s.application.CreateApplicationFromDefinition(ctx, userId, project.Id, definition)
		if err != nil {
			return model.Project{}, err
		}
		if err := mapApplicationDefinition(&maps, definition, created); err != nil {
			return model.Project{}, err
		}
	}
	for _, definition := range item.Services {
		created, err := s.service.CreateServiceFromDefinition(ctx, userId, project.Id, remapServiceDefinition(definition, maps))
		if err != nil {
			return model.Project{}, err
		}
		maps.services[definition.Service.Id] = created.Service.Id
	}
	if item.Gateway != nil {
		created, err := s.gateway.CreateGatewayFromDefinition(ctx, userId, project.Id, *item.Gateway)
		if err != nil {
			return model.Project{}, err
		}
		gatewayServiceId := strings.TrimSpace(item.Gateway.RuntimeService.Service.Id)
		if gatewayServiceId != "" {
			targetGatewayServiceId := strings.TrimSpace(created.RuntimeService.Service.Id)
			if targetGatewayServiceId == "" {
				return model.Project{}, apperror.New(apperror.KindInternal, "Gateway definition restoration returned incomplete RuntimeService")
			}
			maps.services[gatewayServiceId] = targetGatewayServiceId
		}
	}
	for _, definition := range item.Routes {
		if _, err := s.route.CreateRouteFromDefinition(ctx, userId, project.Id, remapRouteDefinition(definition, maps)); err != nil {
			return model.Project{}, err
		}
	}
	return project, nil
}

func mapApplicationDefinition(maps *packageIdMaps, source, target applicationdto.ApplicationDefinition) error {
	if len(source.Versions) != len(target.Versions) {
		return apperror.New(apperror.KindInternal, "Application definition restoration returned incomplete Versions")
	}
	maps.applications[source.Application.Id] = target.Application.Id
	for index, sourceVersion := range source.Versions {
		targetVersion := target.Versions[index]
		if len(sourceVersion.Components) != len(targetVersion.Components) {
			return apperror.New(apperror.KindInternal, "Application definition restoration returned incomplete Components")
		}
		maps.versions[sourceVersion.Version.Id] = targetVersion.Version.Id
		for componentIndex, sourceComponent := range sourceVersion.Components {
			maps.components[sourceComponent.Id] = targetVersion.Components[componentIndex].Id
		}
	}
	return nil
}

func remapServiceDefinition(source servicedto.ServiceDefinition, maps packageIdMaps) servicedto.ServiceDefinition {
	definition := source
	definition.Service.Id = ""
	definition.Service.ProjectId = ""
	definition.Service.ApplicationId = maps.applications[source.Service.ApplicationId]
	definition.Service.VersionId = maps.versions[source.Service.VersionId]
	for index := range definition.Components {
		definition.Components[index].Id = ""
		definition.Components[index].ServiceId = ""
		definition.Components[index].SourceVersionComponentId = maps.components[source.Components[index].SourceVersionComponentId]
	}
	return definition
}

func remapRouteDefinition(source routedto.RouteDefinitionInput, maps packageIdMaps) routedto.RouteDefinitionInput {
	definition := source
	definition.Route.Id = ""
	definition.Route.ProjectId = nil
	definition.Route.Enabled = false
	if source.Route.ServiceId != nil {
		serviceId := maps.services[*source.Route.ServiceId]
		definition.Route.ServiceId = &serviceId
	}
	return definition
}
