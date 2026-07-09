package port

import "context"

type EnvStore interface {
	Load(ctx context.Context) (map[string]string, error)
	Set(ctx context.Context, values map[string]string) error
	Delete(ctx context.Context, keys []string) error
}
