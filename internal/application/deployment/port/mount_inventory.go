package port

import (
	"context"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

// RunningBindMount is a live host directory claim by another service.
type RunningBindMount struct {
	Source string
	Shared bool
}

type MountInventory interface {
	RunningBindMounts(context.Context, environmentport.Target, string) ([]RunningBindMount, error)
}
