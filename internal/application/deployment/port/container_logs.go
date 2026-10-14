package port

import (
	"context"
	"io"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

// ContainerLogRequest selects a service's live containers independently of its workspace.
type ContainerLogRequest struct {
	Project   string
	Component string
	Since     string
	Tail      int
}

// ContainerLogs discovers sources and follows timestamped container output.
// Follow must release its process/session before returning, including on cancellation.
type ContainerLogs interface {
	ContainerIds(context.Context, environmentport.Target, string, string) ([]string, error)
	Follow(context.Context, environmentport.Target, ContainerLogRequest, io.Writer) error
}
