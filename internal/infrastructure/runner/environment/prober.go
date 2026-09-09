package environmentrunner

import (
	"context"
	"errors"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	localrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/local"
	sshrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/ssh"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

// Prober combines the explicit local and SSH prerequisite checks without
// allowing either target type to invoke the other's runtime.
type Prober struct {
	Local *localrunner.Runtime
	SSH   sshrunner.EnvironmentProber
}

func NewProber(local *localrunner.Runtime, ssh sshrunner.EnvironmentProber) Prober {
	return Prober{Local: local, SSH: ssh}
}

func (p Prober) Probe(ctx context.Context, environment model.Environment, privateKey credentialdto.DeploymentSSHPrivateKey) (string, error) {
	if !environment.IsSSH() {
		return "", errors.New("SSH probe received a non-SSH environment")
	}
	return p.SSH.Probe(ctx, environment, privateKey)
}

func (p Prober) ProbeLocal(ctx context.Context, environment model.Environment) error {
	if p.Local == nil {
		return errors.New("local environment probe is not configured")
	}
	return p.Local.ProbeLocal(ctx, environment)
}
