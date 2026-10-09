package port

import "context"

type EnvStore interface {
	Load(ctx context.Context) (map[string]string, error)
	Mutate(ctx context.Context, change func(map[string]string) error) (map[string]string, error)
}
