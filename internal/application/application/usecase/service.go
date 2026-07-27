package applicationsvc

import (
	"context"
	"errors"
	"regexp"
	"strings"

	applicationdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
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

func (s Service) ListApplications(ctx context.Context, userID string, projectID *string, page int, perPage int, search string, kind string) (repository.Page[model.Application], error) {
	if projectID != nil {
		if err := s.ensureProjectMembership(ctx, *projectID, userID); err != nil {
			return repository.Page[model.Application]{}, err
		}
	}
	items, err := s.store.ListApplications(ctx, projectID, page, perPage, search, strings.TrimSpace(kind))
	if err != nil {
		return repository.Page[model.Application]{}, apperror.Wrap(apperror.KindInternal, "Failed to list applications", err)
	}
	return items, nil
}

func (s Service) CreateApplication(ctx context.Context, userID string, input applicationdto.ApplicationCreateInput) (model.Application, error) {
	projectID := strings.TrimSpace(input.ProjectId)
	if projectID == "" {
		return model.Application{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectID, userID); err != nil {
		return model.Application{}, err
	}
	name, code, kind, imagePullPolicy, err := normalizeApplicationCreateInput(input)
	if err != nil {
		return model.Application{}, err
	}
	if err := s.ensureApplicationNameAvailable(ctx, name); err != nil {
		return model.Application{}, err
	}
	app := model.Application{Id: idutil.NewId(), ProjectId: &projectID, Name: name, Code: code, Kind: kind, ImagePullPolicy: imagePullPolicy}
	if err := s.store.CreateApplication(ctx, app); err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to create application", err)
	}
	initialVersion := model.Version{Id: idutil.NewId(), ApplicationId: app.Id, Label: code, Status: status.VersionStatusUnpublished}
	if err := s.store.CreateVersion(ctx, initialVersion); err != nil {
		_ = s.store.DeleteApplication(ctx, app.Id)
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to create initial version", err)
	}
	created, err := s.store.Application(ctx, app.Id)
	if err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	return created, nil
}

func (s Service) ApplicationForUser(ctx context.Context, userID string, applicationID string) (model.Application, error) {
	return s.loadApplicationForUser(ctx, userID, applicationID)
}

func (s Service) UpdateApplication(ctx context.Context, userID string, applicationID string, input applicationdto.ApplicationUpdateInput) (model.Application, error) {
	app, err := s.loadApplicationForUser(ctx, userID, applicationID)
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
	if input.ImagePullPolicy != nil {
		policy := strings.TrimSpace(*input.ImagePullPolicy)
		if !validImagePullPolicy(policy) {
			return model.Application{}, apperror.New(apperror.KindValidation, "Invalid application fields")
		}
		app.ImagePullPolicy = policy
	}
	if err := s.store.UpdateApplication(ctx, app); err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to update application", err)
	}
	updated, err := s.store.Application(ctx, app.Id)
	if err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	return updated, nil
}

func (s Service) loadApplicationForUser(ctx context.Context, userID string, applicationID string) (model.Application, error) {
	applicationID = strings.TrimSpace(applicationID)
	app, err := s.store.Application(ctx, applicationID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Application{}, apperror.New(apperror.KindNotFound, "Application "+applicationID+" not found")
		}
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	if app.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *app.ProjectId, userID); err != nil {
			return model.Application{}, err
		}
	}
	return app, nil
}

func (s Service) ensureProjectMembership(ctx context.Context, projectID string, userID string) error {
	if _, err := s.store.Project(ctx, projectID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Project "+projectID+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.store.IsProjectMember(ctx, projectID, userID)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return nil
}

func (s Service) ensureApplicationNameAvailable(ctx context.Context, name string) error {
	existing, err := s.store.ApplicationByName(ctx, name)
	if err == nil {
		return apperror.New(apperror.KindValidation, "Application '"+existing.Name+"' already exists")
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check application name", err)
	}
	return nil
}

func normalizeApplicationCreateInput(input applicationdto.ApplicationCreateInput) (string, string, string, string, error) {
	name := strings.TrimSpace(input.Name)
	code := strings.TrimSpace(input.Code)
	kind, err := normalizeApplicationKind(input.Kind)
	if err != nil {
		return "", "", "", "", err
	}
	imagePullPolicy := strings.TrimSpace(input.ImagePullPolicy)
	if name == "" || len(name) > 100 || code == "" || len(code) > 100 || !applicationCreateCodePattern.MatchString(code) || !validImagePullPolicy(imagePullPolicy) {
		return "", "", "", "", apperror.New(apperror.KindValidation, "Invalid application fields")
	}
	return name, code, kind, imagePullPolicy, nil
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
