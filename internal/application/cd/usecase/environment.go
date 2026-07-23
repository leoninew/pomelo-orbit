package cdsvc

import (
	"context"
	"errors"
	"regexp"
	"strings"

	cdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

var environmentCodePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func (s Service) ListEnvironments(ctx context.Context, userId string, projectId string, page int, perPage int, search string) (repository.Page[model.Environment], error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return repository.Page[model.Environment]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[model.Environment]{}, err
	}
	items, err := s.store.ListEnvironments(ctx, projectId, page, perPage, search)
	if err != nil {
		return repository.Page[model.Environment]{}, apperror.Wrap(apperror.KindInternal, "Failed to list environments", err)
	}
	return items, nil
}

func (s Service) EnvironmentForUser(ctx context.Context, userId string, environmentId string) (cdto.EnvironmentView, error) {
	env, err := s.loadEnvironmentForUser(ctx, userId, environmentId)
	if err != nil {
		return cdto.EnvironmentView{}, err
	}
	return cdto.EnvironmentView{Environment: env}, nil
}

func (s Service) CreateEnvironment(ctx context.Context, userId string, input cdto.EnvironmentCreateInput) (cdto.EnvironmentView, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return cdto.EnvironmentView{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return cdto.EnvironmentView{}, err
	}
	code := strings.TrimSpace(input.Code)
	name := strings.TrimSpace(input.Name)
	if code == "" || len(code) > 100 || !environmentCodePattern.MatchString(code) || name == "" || len(name) > 255 {
		return cdto.EnvironmentView{}, apperror.New(apperror.KindValidation, "Invalid environment fields")
	}
	env := model.Environment{
		Id:          idutil.NewId(),
		ProjectId:   projectId,
		Code:        code,
		Name:        name,
		Description: normalizeOptionalText(input.Description),
	}
	if err := s.store.CreateEnvironment(ctx, env); err != nil {
		return cdto.EnvironmentView{}, apperror.Wrap(apperror.KindInternal, "Failed to create environment", err)
	}
	return s.EnvironmentForUser(ctx, userId, env.Id)
}

func (s Service) UpdateEnvironment(ctx context.Context, userId string, environmentId string, input cdto.EnvironmentUpdateInput) (cdto.EnvironmentView, error) {
	env, err := s.loadEnvironmentForUser(ctx, userId, environmentId)
	if err != nil {
		return cdto.EnvironmentView{}, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 255 {
			return cdto.EnvironmentView{}, apperror.New(apperror.KindValidation, "Invalid environment name")
		}
		env.Name = name
	}
	if input.Description != nil {
		env.Description = normalizeOptionalText(input.Description)
	}
	if err := s.store.UpdateEnvironment(ctx, env); err != nil {
		return cdto.EnvironmentView{}, apperror.Wrap(apperror.KindInternal, "Failed to update environment", err)
	}
	return s.EnvironmentForUser(ctx, userId, env.Id)
}

func (s Service) DeleteEnvironment(ctx context.Context, userId string, environmentId string) error {
	env, err := s.loadEnvironmentForUser(ctx, userId, environmentId)
	if err != nil {
		return err
	}
	count, err := s.store.CountServicesByEnvironment(ctx, env.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check environment services", err)
	}
	if count > 0 {
		return apperror.New(apperror.KindValidation, "Environment is still referenced by services")
	}
	if err := s.store.DeleteEnvironment(ctx, env.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete environment", err)
	}
	return nil
}

func (s Service) loadEnvironmentForUser(ctx context.Context, userId string, environmentId string) (model.Environment, error) {
	environmentId = strings.TrimSpace(environmentId)
	env, err := s.store.Environment(ctx, environmentId)
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
