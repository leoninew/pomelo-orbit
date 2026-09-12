package dto

import (
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
)

const (
	StatusNeedsEnvironment = "needs_environment"
	StatusNeedsProbe       = "needs_probe"
	StatusNeedsGateway     = "needs_gateway"
	StatusReady            = "ready"
)

type Defaults struct {
	LocalWorkspaceRoot      string
	Image                   string
	RestApiUrl              string
	RestReadyTimeoutSeconds int
	BaseDomain              string
	DefaultEntrypoint       string
	TLSMode                 string
	AcmeProfile             string
	AcmeEmail               string
	DNSApiToken             string
	LocalPlatform           string
	LocalHost               string
	LocalUsername           string
}

type StatusView struct {
	Status      string
	Defaults    Defaults
	Environment *environmentdto.View
	Gateway     *gatewaydto.GatewayView
}

type SaveEnvironmentInput struct {
	TargetType string
	Local      *environmentdto.LocalTargetInput
	SSH        *environmentdto.SSHTargetInput
}

type BootstrapEnvironmentInput struct {
	Username             string
	Password             string
	PrivateKey           string
	PrivateKeyPassphrase string
}

type CreateGatewayInput struct {
	Image                   string
	RestApiUrl              string
	RestReadyTimeoutSeconds int
	BaseDomain              string
	DefaultEntrypoint       string
	TLSMode                 string
	AcmeProfile             string
	AcmeEmail               string
	DNSApiToken             string
}
