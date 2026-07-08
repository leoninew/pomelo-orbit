package projectsvc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"

	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

var projectCodePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

type Repository interface {
	Project(ctx context.Context, id string) (model.Project, error)
	ListProjectsByMember(ctx context.Context, userId string) ([]model.Project, error)
	ListActiveProjectsByMember(ctx context.Context, userId string) ([]model.Project, error)
	ProjectByCode(ctx context.Context, code string) (model.Project, error)
	IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error)
	CreateProject(ctx context.Context, project model.Project, userId string) error
	UpdateProject(ctx context.Context, project model.Project) error
	DeprecateProject(ctx context.Context, projectId string) error
	CountProjectRepositories(ctx context.Context, projectId string) (int, error)
	CountProjectApplications(ctx context.Context, projectId string) (int, error)
	ProjectMembers(ctx context.Context, projectId string) ([]model.User, error)
	AddProjectMember(ctx context.Context, projectId string, userId string) error
	RemoveProjectMember(ctx context.Context, projectId string, userId string) error
}

type UserRepository interface {
	UserById(ctx context.Context, id string) (model.User, error)
}

type Service struct {
	repo  Repository
	users UserRepository
}

type SaveInput struct {
	Name string
	Code string
}

func New(repo Repository, users UserRepository) Service {
	return Service{repo: repo, users: users}
}

func (s Service) ListByMember(ctx context.Context, userId string) ([]model.Project, error) {
	return s.repo.ListProjectsByMember(ctx, userId)
}

func (s Service) Create(ctx context.Context, userId string, input SaveInput) (model.Project, error) {
	name, code, err := normalizeAndValidate(input)
	if err != nil {
		return model.Project{}, err
	}
	if err := s.ensureCodeAvailable(ctx, code, ""); err != nil {
		return model.Project{}, err
	}
	now := time.Now().UTC()
	project := model.Project{Id: idutil.NewId(), Name: name, Code: code, IsActive: true, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.CreateProject(ctx, project, userId); err != nil {
		return model.Project{}, err
	}
	return project, nil
}

func (s Service) LoadForUser(ctx context.Context, projectId string, userId string) (model.Project, error) {
	project, err := s.repo.Project(ctx, projectId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Project{}, apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return model.Project{}, err
	}
	member, err := s.repo.IsProjectMember(ctx, project.Id, userId)
	if err != nil {
		return model.Project{}, apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return model.Project{}, apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return project, nil
}

func (s Service) Update(ctx context.Context, project model.Project, input SaveInput) (model.Project, error) {
	name, code, err := normalizeAndValidate(input)
	if err != nil {
		return model.Project{}, err
	}
	if err := s.ensureCodeAvailable(ctx, code, project.Id); err != nil {
		return model.Project{}, err
	}
	project.Name = name
	project.Code = code
	if err := s.repo.UpdateProject(ctx, project); err != nil {
		return model.Project{}, err
	}
	updated, err := s.repo.Project(ctx, project.Id)
	if err != nil {
		return model.Project{}, err
	}
	return updated, nil
}

func (s Service) Deprecate(ctx context.Context, project model.Project, userId string) error {
	activeProjects, err := s.repo.ListActiveProjectsByMember(ctx, userId)
	if err != nil {
		return fmt.Errorf("list active projects for %s: %w", userId, err)
	}
	if len(activeProjects) <= 1 {
		return apperror.New(apperror.KindValidation, "Cannot deprecate the last active project")
	}
	repoCount, err := s.repo.CountProjectRepositories(ctx, project.Id)
	if err != nil {
		return fmt.Errorf("count project repositories %s: %w", project.Id, err)
	}
	if repoCount > 0 {
		return apperror.New(apperror.KindValidation, fmt.Sprintf("Cannot deprecate project with %d repositories", repoCount))
	}
	appCount, err := s.repo.CountProjectApplications(ctx, project.Id)
	if err != nil {
		return fmt.Errorf("count project applications %s: %w", project.Id, err)
	}
	if appCount > 0 {
		return apperror.New(apperror.KindValidation, fmt.Sprintf("Cannot deprecate project with %d applications", appCount))
	}
	return s.repo.DeprecateProject(ctx, project.Id)
}

func (s Service) Members(ctx context.Context, projectId string) ([]model.User, error) {
	return s.repo.ProjectMembers(ctx, projectId)
}

func (s Service) AddMember(ctx context.Context, projectId string, userId string) ([]model.User, error) {
	userId = strings.TrimSpace(userId)
	if userId == "" {
		return nil, apperror.New(apperror.KindValidation, "user_id is required")
	}
	if _, err := s.users.UserById(ctx, userId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.New(apperror.KindNotFound, "User "+userId+" not found")
		}
		return nil, fmt.Errorf("load member user %s: %w", userId, err)
	}
	if err := s.repo.AddProjectMember(ctx, projectId, userId); err != nil {
		return nil, err
	}
	return s.repo.ProjectMembers(ctx, projectId)
}

func (s Service) RemoveMember(ctx context.Context, projectId string, userId string) ([]model.User, error) {
	if err := s.repo.RemoveProjectMember(ctx, projectId, strings.TrimSpace(userId)); err != nil {
		return nil, err
	}
	return s.repo.ProjectMembers(ctx, projectId)
}

func (s Service) ensureCodeAvailable(ctx context.Context, code string, currentProjectId string) error {
	existing, err := s.repo.ProjectByCode(ctx, code)
	if err == nil {
		if existing.Id != currentProjectId {
			return apperror.New(apperror.KindConflict, "Project code "+code+" already exists")
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check project code %s: %w", code, err)
	}
	return nil
}

func normalizeAndValidate(input SaveInput) (string, string, error) {
	name := strings.TrimSpace(input.Name)
	code := strings.TrimSpace(input.Code)
	if name == "" || len(name) > 100 || code == "" || len(code) > 100 || !projectCodePattern.MatchString(code) {
		return "", "", ErrInvalidProjectFields
	}
	return name, code, nil
}

var ErrInvalidProjectFields = apperror.New(apperror.KindValidation, "Invalid project fields")
