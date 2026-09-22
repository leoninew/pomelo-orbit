package applicationsvc

import (
	"context"
	"errors"
	"regexp"
	"strings"

	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

var applicationCreateCodePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var applicationUpdateCodePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// Service owns application and version specifications only.
type Service struct {
	store *stores
}

func New(project repository.ProjectReader, application repository.ApplicationStore, services ...repository.ServiceReader) Service {
	store := &stores{project: project, application: application}
	if len(services) > 0 {
		store.service = services[0]
	}
	return Service{store: store}
}

func (s Service) ListApplications(ctx context.Context, userId string, projectId string, page int, perPage int, search string, kind string) (repository.Page[model.Application], error) {
	if projectId == "" {
		return repository.Page[model.Application]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[model.Application]{}, err
	}
	items, err := s.store.ListApplications(ctx, projectId, page, perPage, search, strings.TrimSpace(kind))
	if err != nil {
		return repository.Page[model.Application]{}, apperror.Wrap(apperror.KindInternal, "Failed to list applications", err)
	}
	return items, nil
}

func (s Service) CreateApplication(ctx context.Context, userId string, input applicationdto.ApplicationCreateInput) (model.Application, error) {
	projectId := input.ProjectId
	if projectId == "" {
		return model.Application{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.Application{}, err
	}
	name, code, kind, err := normalizeApplicationCreateInput(input)
	if err != nil {
		return model.Application{}, err
	}
	if err := s.ensureApplicationNameAvailable(ctx, projectId, name); err != nil {
		return model.Application{}, err
	}
	if err := s.ensureApplicationCodeAvailable(ctx, projectId, code); err != nil {
		return model.Application{}, err
	}
	app := model.Application{Id: idutil.NewId(), ProjectId: &projectId, Name: name, Code: code, Kind: kind}
	if err := s.store.CreateApplication(ctx, app); err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to create application", err)
	}
	initialVersion := model.Version{Id: idutil.NewId(), ApplicationId: app.Id, Label: code, Status: status.VersionStatusUnpublished}
	if err := s.store.CreateVersion(ctx, projectId, initialVersion); err != nil {
		_ = s.store.DeleteApplication(ctx, projectId, app.Id)
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to create initial version", err)
	}
	created, err := s.store.Application(ctx, projectId, app.Id)
	if err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	return created, nil
}

func (s Service) ApplicationForUser(ctx context.Context, userId string, projectId string, applicationId string) (model.Application, error) {
	return s.loadApplicationForUser(ctx, userId, projectId, applicationId)
}

// ApplicationDefinitionForUser returns an Application with all of its Version
// and Component definitions. It does not include runtime Service bindings.
func (s Service) ApplicationDefinitionForUser(ctx context.Context, userId, projectId, applicationId string) (applicationdto.ApplicationDefinition, error) {
	app, err := s.loadApplicationForUser(ctx, userId, projectId, applicationId)
	if err != nil {
		return applicationdto.ApplicationDefinition{}, err
	}
	versions, err := s.store.ListVersions(ctx, projectId, app.Id)
	if err != nil {
		return applicationdto.ApplicationDefinition{}, apperror.Wrap(apperror.KindInternal, "Failed to list application versions", err)
	}
	definition := applicationdto.ApplicationDefinition{Application: app, Versions: make([]applicationdto.VersionDefinition, 0, len(versions))}
	for _, version := range versions {
		components, err := s.store.VersionComponentsByVersion(ctx, projectId, version.Id)
		if err != nil {
			return applicationdto.ApplicationDefinition{}, apperror.Wrap(apperror.KindInternal, "Failed to list version components", err)
		}
		definition.Versions = append(definition.Versions, applicationdto.VersionDefinition{Version: version, Components: components})
	}
	return definition, nil
}

// CreateApplicationFromDefinition creates the complete static Application
// definition without synthesizing an interactive initial Version.
func (s Service) CreateApplicationFromDefinition(ctx context.Context, userId, projectId string, input applicationdto.ApplicationDefinition) (applicationdto.ApplicationDefinition, error) {
	if projectId == "" {
		return applicationdto.ApplicationDefinition{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return applicationdto.ApplicationDefinition{}, err
	}
	name, code, kind, err := normalizeApplicationCreateInput(applicationdto.ApplicationCreateInput{
		ProjectId: projectId, Name: input.Application.Name, Code: input.Application.Code, Kind: input.Application.Kind,
	})
	if err != nil {
		return applicationdto.ApplicationDefinition{}, err
	}
	if err := s.ensureApplicationNameAvailable(ctx, projectId, name); err != nil {
		return applicationdto.ApplicationDefinition{}, err
	}
	if err := s.ensureApplicationCodeAvailable(ctx, projectId, code); err != nil {
		return applicationdto.ApplicationDefinition{}, err
	}
	app := model.Application{Id: idutil.NewId(), ProjectId: &projectId, Name: name, Code: code, Kind: kind}
	if err := s.store.CreateApplication(ctx, app); err != nil {
		return applicationdto.ApplicationDefinition{}, apperror.Wrap(apperror.KindInternal, "Failed to create application", err)
	}
	createdVersions, err := s.createDefinitionVersions(ctx, projectId, app, input.Versions)
	if err != nil {
		return applicationdto.ApplicationDefinition{}, err
	}
	return applicationdto.ApplicationDefinition{Application: app, Versions: createdVersions}, nil
}

func (s Service) UpdateApplication(ctx context.Context, userId string, projectId string, applicationId string, input applicationdto.ApplicationUpdateInput) (model.Application, error) {
	app, err := s.loadApplicationForUser(ctx, userId, projectId, applicationId)
	if err != nil {
		return model.Application{}, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 100 {
			return model.Application{}, apperror.New(apperror.KindValidation, "Invalid application fields")
		}
		app.Name = name
	}
	if input.Code != nil {
		code := strings.TrimSpace(*input.Code)
		if code == "" || len(code) > 100 || !applicationUpdateCodePattern.MatchString(code) {
			return model.Application{}, apperror.New(apperror.KindValidation, "Invalid application fields")
		}
		app.Code = code
	}
	if err := s.ensureApplicationCodeAvailableExcept(ctx, projectId, app.Code, app.Id); err != nil {
		return model.Application{}, err
	}
	if err := s.store.UpdateApplication(ctx, projectId, app); err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to update application", err)
	}
	updated, err := s.store.Application(ctx, projectId, app.Id)
	if err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	return updated, nil
}

// RemoveApplication removes a static Application definition after its runtime
// Service bindings have been removed by the Service domain.
func (s Service) RemoveApplication(ctx context.Context, userId, projectId, applicationId string) error {
	app, err := s.loadApplicationForUser(ctx, userId, projectId, applicationId)
	if err != nil {
		return err
	}
	if s.store.service == nil {
		return apperror.New(apperror.KindInternal, "application service reader is not configured")
	}
	services, err := s.store.service.ListServicesByApplication(ctx, projectId, app.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to list application services", err)
	}
	if len(services) > 0 {
		return apperror.New(apperror.KindValidation, "Application still has service bindings")
	}
	if err := s.store.DeleteApplication(ctx, projectId, app.Id); err != nil {
		if errors.Is(err, repository.ErrReferenced) {
			return apperror.New(apperror.KindValidation, "Application is still referenced and cannot be deleted")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to remove application", err)
	}
	return nil
}

func (s Service) loadApplicationForUser(ctx context.Context, userId string, projectId string, applicationId string) (model.Application, error) {
	if projectId == "" {
		return model.Application{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.Application{}, err
	}
	app, err := s.store.Application(ctx, projectId, applicationId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Application{}, apperror.New(apperror.KindNotFound, "Application "+applicationId+" not found")
		}
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	return app, nil
}

func (s Service) ensureProjectMembership(ctx context.Context, projectId string, userId string) error {
	if _, err := s.store.Project(ctx, projectId); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.store.IsProjectMember(ctx, projectId, userId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return nil
}

func (s Service) ensureApplicationNameAvailable(ctx context.Context, projectId string, name string) error {
	existing, err := s.store.ApplicationByProjectAndName(ctx, projectId, name)
	if err == nil {
		return apperror.New(apperror.KindValidation, "Application '"+existing.Name+"' already exists")
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check application name", err)
	}
	return nil
}

func (s Service) ensureApplicationCodeAvailable(ctx context.Context, projectId string, code string) error {
	return s.ensureApplicationCodeAvailableExcept(ctx, projectId, code, "")
}

func (s Service) ensureApplicationCodeAvailableExcept(ctx context.Context, projectId, code, exceptId string) error {
	existing, err := s.store.ApplicationByProjectAndCode(ctx, projectId, code)
	if err == nil && existing.Id != exceptId {
		return apperror.New(apperror.KindValidation, "Application code '"+existing.Code+"' already exists")
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check application code", err)
	}
	return nil
}

func normalizeApplicationCreateInput(input applicationdto.ApplicationCreateInput) (string, string, string, error) {
	name := strings.TrimSpace(input.Name)
	code := strings.TrimSpace(input.Code)
	kind, err := normalizeApplicationKind(input.Kind)
	if err != nil {
		return "", "", "", err
	}
	if name == "" || len(name) > 100 || code == "" || len(code) > 100 || !applicationCreateCodePattern.MatchString(code) {
		return "", "", "", apperror.New(apperror.KindValidation, "Invalid application fields")
	}
	return name, code, kind, nil
}

func normalizeApplicationKind(kind string) (string, error) {
	switch strings.TrimSpace(kind) {
	case "":
		return status.ApplicationKindStandard, nil
	case status.ApplicationKindStandard, status.ApplicationKindGateway:
		return strings.TrimSpace(kind), nil
	default:
		return "", apperror.New(apperror.KindValidation, "kind must be standard or gateway")
	}
}

func validImagePullPolicy(value string) bool {
	return value == "always" || value == "missing" || value == "never"
}
