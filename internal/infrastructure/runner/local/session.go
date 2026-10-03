package localrunner

import (
	"context"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

func (r *Runtime) OpenSession(ctx context.Context, _ environmentport.Target) (context.Context, func(), error) {
	return ctx, func() {}, ctx.Err()
}
