package repository

import (
	"context"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// AuthStore persists authentication audit records.
type AuthStore interface {
	SaveLoginHistory(ctx context.Context, history model.LoginHistory) error
	ListLoginHistory(ctx context.Context, page int, perPage int, search string) (Page[model.LoginHistory], error)
}
