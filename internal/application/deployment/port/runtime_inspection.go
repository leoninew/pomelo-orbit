package port

import (
	"context"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

type RuntimeScope struct {
	Target   environmentport.Target
	Location ServiceLocation
	Project  string
}

type ContainerObservation struct {
	Containers []deploymentdto.RuntimeContainer
	Raw        any
}

type ContainerInspection struct {
	Raw          any
	Networks     []string
	State        string
	Health       string
	RestartCount int
}

// ContainerObserver returns runtime facts without exposing command or JSON protocols.
type ContainerObserver interface {
	Containers(context.Context, RuntimeScope) (ContainerObservation, error)
}

type RuntimeInspector interface {
	ContainerObserver
	ConfigurationServices(string) (map[string]string, error)
	Config(context.Context, RuntimeScope) (string, error)
	Logs(context.Context, RuntimeScope, int, string, []string) (string, error)
	InspectContainer(context.Context, RuntimeScope, string) (ContainerInspection, error)
	InspectNetwork(context.Context, RuntimeScope, string) (any, error)
	HTTPProbe(context.Context, RuntimeScope, string, int, string) (string, error)
	Version(context.Context, RuntimeScope) (string, error)
}
