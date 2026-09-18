package dto

import (
	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

type GatewayCreateInput struct {
	ProjectId               string
	Code                    string
	Name                    string
	RestApiUrl              string
	RestApiHostUrl          string
	RestReadyTimeoutSeconds *int
	BaseDomain              string
	InitialComponentImage   *string
	DefaultEntrypoint       *string
	TLSMode                 *string
	AcmeProfile             *string
	AcmeEmail               *string
	DNSApiToken             *string
}

type GatewayUpdateInput struct {
	Name                    *string
	RestApiUrl              *string
	RestApiHostUrl          *string
	RestReadyTimeoutSeconds *int
	BaseDomain              *string
	DefaultEntrypoint       *string
	TLSMode                 *string
	AcmeProfile             *string
	AcmeEmail               *string
	DNSApiToken             *string
}

// GatewayDefinition is the complete saved configuration of the managed
// Gateway. It owns its Application, Versions, runtime Service, Config,
// profile bindings, and Environment binding.
type GatewayDefinition struct {
	Application    applicationdto.ApplicationDefinition
	Config         model.GatewayConfig
	RuntimeService servicedto.ServiceDefinition
}

// GatewayExposureItem is an active local or public application exposure.
type GatewayExposureItem struct {
	ApplicationId   string
	ApplicationCode string
	ComponentName   string
	Protocol        string
	Access          string
	ContainerPort   int
	ListenPort      int
	PublicHost      string
	InternalDns     string
	ClientHint      string
}

type GatewayView struct {
	Application model.Application
	Config      model.GatewayConfig
	Service     *model.Service
	Exposures   []GatewayExposureItem
}

type ProvisionGatewayInput struct {
	ProjectId string
}

type ProvisionGatewayResult struct {
	GatewayCreated bool
	ServiceCreated bool
	Gateway        GatewayView
	Service        model.Service
	Steps          []string
}
