package cdsvc

import (
	"context"
	"encoding/json"
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
	bindings, err := s.store.BindingsByEnvironment(ctx, env.Id)
	if err != nil {
		return cdto.EnvironmentView{}, apperror.Wrap(apperror.KindInternal, "Failed to list bindings", err)
	}
	return cdto.EnvironmentView{Environment: env, Bindings: bindings}, nil
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

func (s Service) ReplaceEnvironmentBindings(ctx context.Context, userId string, environmentId string, inputs []cdto.EnvironmentBindingInput) (cdto.EnvironmentView, error) {
	env, err := s.loadEnvironmentForUser(ctx, userId, environmentId)
	if err != nil {
		return cdto.EnvironmentView{}, err
	}
	bindings, err := normalizeBindings(env.Id, inputs)
	if err != nil {
		return cdto.EnvironmentView{}, err
	}
	if err := validateBindingConflicts(bindings); err != nil {
		return cdto.EnvironmentView{}, apperror.New(apperror.KindValidation, err.Error())
	}
	if err := s.store.ReplaceBindings(ctx, env.Id, bindings); err != nil {
		return cdto.EnvironmentView{}, apperror.Wrap(apperror.KindInternal, "Failed to replace bindings", err)
	}
	return s.EnvironmentForUser(ctx, userId, env.Id)
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

func normalizeBindings(environmentId string, inputs []cdto.EnvironmentBindingInput) ([]model.EnvironmentBinding, error) {
	bindings := make([]model.EnvironmentBinding, 0, len(inputs))
	for _, input := range inputs {
		componentName := strings.TrimSpace(input.ComponentName)
		protocol := strings.ToLower(strings.TrimSpace(input.Protocol))
		entrypoint := strings.TrimSpace(input.Entrypoint)
		tlsMode := strings.ToLower(strings.TrimSpace(input.TLSMode))
		if tlsMode == "" {
			tlsMode = "none"
		}
		if componentName == "" || (protocol != "http" && protocol != "tcp") || input.ContainerPort < 1 || input.ContainerPort > 65535 {
			return nil, apperror.New(apperror.KindValidation, "Invalid binding fields")
		}
		if protocol == "http" && entrypoint == "" {
			entrypoint = "websecure"
		}
		if entrypoint == "" {
			return nil, apperror.New(apperror.KindValidation, "entrypoint is required")
		}
		if tlsMode != "none" && tlsMode != "letsencrypt" && tlsMode != "manual" {
			return nil, apperror.New(apperror.KindValidation, "Invalid tls_mode")
		}
		domains := make([]string, 0, len(input.Domains))
		for _, domain := range input.Domains {
			domain = strings.ToLower(strings.TrimSpace(domain))
			if domain == "" {
				continue
			}
			if !validDeploymentRouteDomain(domain) {
				return nil, apperror.New(apperror.KindValidation, "Invalid binding domain")
			}
			domains = append(domains, domain)
		}
		if protocol == "http" && len(domains) == 0 {
			return nil, apperror.New(apperror.KindValidation, "http binding requires at least one domain")
		}
		raw, err := json.Marshal(domains)
		if err != nil {
			return nil, apperror.Wrap(apperror.KindInternal, "Failed to encode domains", err)
		}
		bindings = append(bindings, model.EnvironmentBinding{
			Id:            idutil.NewId(),
			EnvironmentId: environmentId,
			ComponentName: componentName,
			Protocol:      protocol,
			ContainerPort: input.ContainerPort,
			DomainsJSON:   string(raw),
			Entrypoint:    entrypoint,
			TLSMode:       tlsMode,
			SNIHost:       normalizeOptionalText(input.SNIHost),
			Note:          normalizeOptionalText(input.Note),
		})
	}
	return bindings, nil
}

func validateBindingConflicts(bindings []model.EnvironmentBinding) error {
	httpHosts := map[string]struct{}{}
	tcpKeys := map[string]struct{}{}
	for _, binding := range bindings {
		if binding.Protocol == "http" {
			domains, err := parseStringSliceJSON(&binding.DomainsJSON)
			if err != nil {
				return err
			}
			for _, domain := range domains {
				key := domain
				if _, ok := httpHosts[key]; ok {
					return errors.New("duplicate http domain binding: " + domain)
				}
				httpHosts[key] = struct{}{}
			}
			continue
		}
		sni := ""
		if binding.SNIHost != nil {
			sni = strings.TrimSpace(*binding.SNIHost)
		}
		key := binding.Entrypoint + "|" + sni
		if _, ok := tcpKeys[key]; ok {
			return errors.New("duplicate tcp entrypoint/sni binding")
		}
		tcpKeys[key] = struct{}{}
	}
	return nil
}
