package port

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

type ProjectAccessReader interface {
	ListProjectsByMember(context.Context, string) ([]model.Project, error)
	ProjectByCode(context.Context, string) (model.Project, error)
	IsProjectMember(context.Context, string, string) (bool, error)
}
