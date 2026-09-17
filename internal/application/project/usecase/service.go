package projectsvc

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	projectdto "github.com/leoninew/pomelo-orbit/internal/application/project/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

var projectCodePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

type Service struct {
	repo         repository.ProjectStore
	users        repository.UserStore
	repositories repository.RepositoryProjectCounter
	applications repository.ApplicationProjectCounter
}

func New(
	repo repository.ProjectStore,
	users repository.UserStore,
	repositories repository.RepositoryProjectCounter,
	applications repository.ApplicationProjectCounter,
) Service {
	return Service{repo: repo, users: users, repositories: repositories, applications: applications}
}

func (s Service) ListByMember(ctx context.Context, userId string) ([]model.Project, error) {
	return s.repo.ListProjectsByMember(ctx, userId)
}

func (s Service) Create(ctx context.Context, userId string, input projectdto.CreateInput) (model.Project, error) {
	name, code, err := normalizeAndValidateCreate(input)
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

// CreateFromDefinition creates a Project and assigns the initiating user as
// its initial member.
func (s Service) CreateFromDefinition(ctx context.Context, userId string, input projectdto.ProjectDefinition) (model.Project, error) {
	name, code, err := normalizeProjectDefinition(input)
	if err != nil {
		return model.Project{}, err
	}
	if err := s.ensureCodeAvailable(ctx, code, ""); err != nil {
		return model.Project{}, err
	}
	now := time.Now().UTC()
	project := model.Project{Id: idutil.NewId(), Name: name, Code: code, IsActive: input.IsActive, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.CreateProject(ctx, project, userId); err != nil {
		return model.Project{}, err
	}
	return project, nil
}

func (s Service) LoadForUser(ctx context.Context, projectId string, userId string) (model.Project, error) {
	project, err := s.repo.Project(ctx, projectId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
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

func (s Service) Update(ctx context.Context, project model.Project, input projectdto.SaveInput) (model.Project, error) {
	name, err := normalizeAndValidateUpdate(input)
	if err != nil {
		return model.Project{}, err
	}
	project.Name = name
	if err := s.repo.UpdateProject(ctx, project); err != nil {
		return model.Project{}, err
	}
	updated, err := s.repo.Project(ctx, project.Id)
	if err != nil {
		return model.Project{}, err
	}
	return updated, nil
}

// UpdateFromDefinition replaces a Project's mutable business identity while
// preserving its ID and member relationships.
func (s Service) UpdateFromDefinition(ctx context.Context, userId, projectId string, input projectdto.ProjectDefinition) (model.Project, error) {
	project, err := s.LoadForUser(ctx, projectId, userId)
	if err != nil {
		return model.Project{}, err
	}
	name, code, err := normalizeProjectDefinition(input)
	if err != nil {
		return model.Project{}, err
	}
	if err := s.ensureCodeAvailable(ctx, code, project.Id); err != nil {
		return model.Project{}, err
	}
	project.Name = name
	project.Code = code
	project.IsActive = input.IsActive
	if err := s.repo.UpdateProject(ctx, project); err != nil {
		return model.Project{}, err
	}
	return s.repo.Project(ctx, project.Id)
}

func (s Service) Deprecate(ctx context.Context, project model.Project, userId string) error {
	activeProjects, err := s.repo.ListActiveProjectsByMember(ctx, userId)
	if err != nil {
		return fmt.Errorf("list active projects for %s: %w", userId, err)
	}
	if len(activeProjects) <= 1 {
		return apperror.New(apperror.KindValidation, "Cannot deprecate the last active project")
	}
	repoCount, err := s.repositories.CountRepositoriesByProject(ctx, project.Id)
	if err != nil {
		return fmt.Errorf("count project repositories %s: %w", project.Id, err)
	}
	if repoCount > 0 {
		return apperror.New(apperror.KindValidation, fmt.Sprintf("Cannot deprecate project with %d repositories", repoCount))
	}
	appCount, err := s.applications.CountApplicationsByProject(ctx, project.Id)
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
		if errors.Is(err, repository.ErrNotFound) {
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

// RemoveUserFromAllProjects removes Project-owned memberships before the User
// domain deletes the account itself.
func (s Service) RemoveUserFromAllProjects(ctx context.Context, userId string) error {
	return s.repo.RemoveUserFromAllProjects(ctx, strings.TrimSpace(userId))
}

func (s Service) ensureCodeAvailable(ctx context.Context, code string, currentProjectId string) error {
	existing, err := s.repo.ProjectByCode(ctx, code)
	if err == nil {
		if existing.Id != currentProjectId {
			return apperror.New(apperror.KindConflict, "Project code "+code+" already exists")
		}
		return nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("check project code %s: %w", code, err)
	}
	return nil
}

func normalizeAndValidateCreate(input projectdto.CreateInput) (string, string, error) {
	name := strings.TrimSpace(input.Name)
	code := strings.TrimSpace(input.Code)
	if name == "" || len(name) > 100 || code == "" || len(code) > 100 || !projectCodePattern.MatchString(code) {
		return "", "", ErrInvalidProjectFields
	}
	return name, code, nil
}

func normalizeAndValidateUpdate(input projectdto.SaveInput) (string, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 100 {
		return "", ErrInvalidProjectFields
	}
	return name, nil
}

func normalizeProjectDefinition(input projectdto.ProjectDefinition) (string, string, error) {
	return normalizeAndValidateCreate(projectdto.CreateInput{Name: input.Name, Code: input.Code})
}

var ErrInvalidProjectFields = apperror.New(apperror.KindValidation, "Invalid project fields")
