package environmentsvc

import (
	"context"
	"errors"
	"sort"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type AccessService struct {
	environments Service
	projects     environmentport.ProjectAccessReader
	terminal     *TerminalService
}

func NewAccessService(environments Service, projects environmentport.ProjectAccessReader, terminal *TerminalService) *AccessService {
	return &AccessService{environments: environments, projects: projects, terminal: terminal}
}

func (s *AccessService) List(ctx context.Context, actorId string) ([]environmentdto.ProjectSummary, error) {
	projects, err := s.projects.ListProjectsByMember(ctx, actorId)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list projects", err)
	}
	result := make([]environmentdto.ProjectSummary, 0, len(projects))
	for _, p := range projects {
		item := environmentdto.ProjectSummary{Id: p.Id, Name: p.Name, Code: p.Code, IsActive: p.IsActive, TargetType: "unconfigured"}
		e, err := s.environments.environments.EnvironmentByProject(ctx, p.Id)
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.Wrap(apperror.KindInternal, "Failed to load project environment", err)
		}
		if err == nil {
			item.EnvironmentId, item.TargetType = e.Id, e.TargetType
			if e.IsSSH() {
				item.Platform, item.Host, item.Port, item.Username = e.SSH.Platform, e.SSH.Host, e.SSH.Port, e.SSH.Username
			}
			if e.IsLocal() {
				d := s.environments.localDisplay
				item.Platform, item.Host, item.Username = d.Platform, d.Host, d.Username
			}
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	return result, nil
}

func (s *AccessService) project(ctx context.Context, actorId, code string) (model.Project, error) {
	if code == "" {
		return model.Project{}, apperror.New(apperror.KindValidation, "project_code is required")
	}
	p, err := s.projects.ProjectByCode(ctx, code)
	if errors.Is(err, repository.ErrNotFound) {
		return p, apperror.New(apperror.KindNotFound, "Project not found")
	}
	if err != nil {
		return p, apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.projects.IsProjectMember(ctx, p.Id, actorId)
	if err != nil {
		return p, apperror.Wrap(apperror.KindInternal, "Failed to check project membership", err)
	}
	if !member {
		return p, apperror.New(apperror.KindForbidden, "Not a project member")
	}
	return p, nil
}

func (s *AccessService) target(ctx context.Context, actorId, code string) (model.Project, environmentport.Target, error) {
	p, err := s.project(ctx, actorId, code)
	if err != nil {
		return p, environmentport.Target{}, err
	}
	target, err := s.environments.resolveConnectionTarget(ctx, p.Id)
	if apperror.IsKind(err, apperror.KindNotFound) {
		err = apperror.New(apperror.KindValidation, "Project environment is not configured")
	}
	if err == nil && !target.Environment.IsSSH() {
		err = apperror.New(apperror.KindValidation, "Project environment must be SSH")
	}
	return p, target, err
}

func (s *AccessService) AcquireTerminal(ctx context.Context, actorId, code string) (TerminalGrant, func(), error) {
	p, _, err := s.target(ctx, actorId, code)
	if err != nil {
		return TerminalGrant{}, nil, err
	}
	grant, release, err := s.terminal.Acquire(ctx, actorId, p.Id)
	if err != nil {
		return grant, nil, err
	}
	if !grant.Target.Environment.IsSSH() {
		release()
		return TerminalGrant{}, nil, apperror.New(apperror.KindValidation, "Project environment must be SSH")
	}
	return grant, release, nil
}
