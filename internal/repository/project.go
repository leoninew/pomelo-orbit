package repository

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// ProjectReader is the project membership read surface shared by downstream domains.
type ProjectReader interface {
	Project(ctx context.Context, id string) (model.Project, error)
	IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error)
}

// ProjectStore persists projects and project membership.
type ProjectStore interface {
	ProjectReader
	ListProjectsByMember(ctx context.Context, userId string) ([]model.Project, error)
	ListActiveProjectsByMember(ctx context.Context, userId string) ([]model.Project, error)
	ProjectByCode(ctx context.Context, code string) (model.Project, error)
	CreateProject(ctx context.Context, project model.Project, userId string) error
	UpdateProject(ctx context.Context, project model.Project) error
	DeprecateProject(ctx context.Context, projectId string) error
	CountProjectRepositories(ctx context.Context, projectId string) (int, error)
	CountProjectApplications(ctx context.Context, projectId string) (int, error)
	ProjectMembers(ctx context.Context, projectId string) ([]model.User, error)
	AddProjectMember(ctx context.Context, projectId string, userId string) error
	RemoveProjectMember(ctx context.Context, projectId string, userId string) error
	RemoveUserFromAllProjects(ctx context.Context, userId string) error
}
