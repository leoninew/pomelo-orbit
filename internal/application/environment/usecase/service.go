package environmentsvc

import (
	"context"
	"errors"
	"regexp"
	"strings"

	environmentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/environment/dto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

var environmentCodePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

type Service struct {
	project     repository.ProjectReader
	environment repository.EnvironmentStore
}

func New(project repository.ProjectReader, environment repository.EnvironmentStore) Service {
	return Service{project: project, environment: environment}
}

func (s Service) ListEnvironments(ctx context.Context, userId string, projectId string, page int, perPage int, search string) (repository.Page[model.Environment], error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return repository.Page[model.Environment]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[model.Environment]{}, err
	}
	items, err := s.environment.ListEnvironments(ctx, projectId, page, perPage, search)
	if err != nil {
		return repository.Page[model.Environment]{}, apperror.Wrap(apperror.KindInternal, "Failed to list environments", err)
	}
	return items, nil
}

func (s Service) EnvironmentForUser(ctx context.Context, userId string, environmentId string) (environmentdto.EnvironmentView, error) {
	env, err := s.loadEnvironmentForUser(ctx, userId, environmentId)
	if err != nil {
		return environmentdto.EnvironmentView{}, err
	}
	return environmentdto.EnvironmentView{Environment: env}, nil
}

func (s Service) CreateEnvironment(ctx context.Context, userId string, input environmentdto.EnvironmentCreateInput) (environmentdto.EnvironmentView, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return environmentdto.EnvironmentView{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return environmentdto.EnvironmentView{}, err
	}
	code := strings.TrimSpace(input.Code)
	name := strings.TrimSpace(input.Name)
	if code == "" || len(code) > 100 || !environmentCodePattern.MatchString(code) || name == "" || len(name) > 255 {
		return environmentdto.EnvironmentView{}, apperror.New(apperror.KindValidation, "Invalid environment fields")
	}
	env := model.Environment{
		Id:          idutil.NewId(),
		ProjectId:   projectId,
		Code:        code,
		Name:        name,
		Description: normalizeOptionalText(input.Description),
	}
	if err := s.environment.CreateEnvironment(ctx, env); err != nil {
		return environmentdto.EnvironmentView{}, apperror.Wrap(apperror.KindInternal, "Failed to create environment", err)
	}
	return s.EnvironmentForUser(ctx, userId, env.Id)
}

func (s Service) UpdateEnvironment(ctx context.Context, userId string, environmentId string, input environmentdto.EnvironmentUpdateInput) (environmentdto.EnvironmentView, error) {
	env, err := s.loadEnvironmentForUser(ctx, userId, environmentId)
	if err != nil {
		return environmentdto.EnvironmentView{}, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 255 {
			return environmentdto.EnvironmentView{}, apperror.New(apperror.KindValidation, "Invalid environment name")
		}
		env.Name = name
	}
	if input.Description != nil {
		env.Description = normalizeOptionalText(input.Description)
	}
	if err := s.environment.UpdateEnvironment(ctx, env); err != nil {
		return environmentdto.EnvironmentView{}, apperror.Wrap(apperror.KindInternal, "Failed to update environment", err)
	}
	return s.EnvironmentForUser(ctx, userId, env.Id)
}

func (s Service) DeleteEnvironment(ctx context.Context, userId string, environmentId string) error {
	env, err := s.loadEnvironmentForUser(ctx, userId, environmentId)
	if err != nil {
		return err
	}
	count, err := s.environment.CountServicesByEnvironment(ctx, env.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check environment services", err)
	}
	if count > 0 {
		return apperror.New(apperror.KindValidation, "Environment is still referenced by services")
	}
	if err := s.environment.DeleteEnvironment(ctx, env.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete environment", err)
	}
	return nil
}

func (s Service) loadEnvironmentForUser(ctx context.Context, userId string, environmentId string) (model.Environment, error) {
	environmentId = strings.TrimSpace(environmentId)
	env, err := s.environment.Environment(ctx, environmentId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Environment{}, apperror.New(apperror.KindNotFound, "Environment "+environmentId+" not found")
		}
		return model.Environment{}, apperror.Wrap(apperror.KindInternal, "Failed to load environment", err)
	}
	if err := s.ensureProjectMembership(ctx, env.ProjectId, userId); err != nil {
		return model.Environment{}, err
	}
	return env, nil
}

func (s Service) ensureProjectMembership(ctx context.Context, projectId string, userId string) error {
	if _, err := s.project.Project(ctx, projectId); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
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

func normalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
