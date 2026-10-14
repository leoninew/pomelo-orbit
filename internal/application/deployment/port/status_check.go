package port

import (
	"context"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

// ContainerState is an observed fact; service status remains an application rule.
type ContainerState struct {
	State     string
	Unhealthy bool
}

type ContainerInventory interface {
	Containers(context.Context, environmentport.Target) (map[string][]ContainerState, error)
}

type StatusCheckStore interface {
	DirectoryServices(context.Context, string) ([]model.Service, error)
	UpdateServiceObservedStatus(context.Context, model.Service, string, int64, string) (bool, error)
}
