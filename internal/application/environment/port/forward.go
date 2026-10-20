package port

import (
	"context"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
)

// OpenForward authenticates SSH. Listen is called only after business confirms
// the observed host key and rechecks the connection's access and target revision.
type ForwardRunner interface {
	OpenForward(context.Context, Target, int, int) (ForwardSession, error)
}

type ForwardSession interface {
	HostKeyFingerprint() string
	Listen() error
	Wait() error
	Close() error
}

// ForwardProcesses manages only this OS user's local background processes.
// Status and Stop do not require a business database or remote credentials.
type ForwardProcesses interface {
	Start(context.Context, environmentdto.ForwardSpec) (environmentdto.ForwardStatus, error)
	Status(context.Context, int) ([]environmentdto.ForwardStatus, error)
	// Stop reports whether a managed record was found and cleaned up.
	// An absent record returns false, nil; repeated stops remain successful.
	Stop(context.Context, int) (bool, error)
}
