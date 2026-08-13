package dto

import (
	"time"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

type GatewayCreateInput struct {
	ProjectId                  string
	Code                       string
	Name                       string
	RestApiUrl                 string
	BaseDomain                 string
	InitialComponentImage      *string
	InitialComponentPullPolicy string
	DefaultEntrypoint          *string
	TLSMode                    *string
}

// GatewayCreateDefaults is the process-level default set for managed gateway create.
type GatewayCreateDefaults struct {
	Code                       string
	Name                       string
	RestApiUrl                 string
	BaseDomain                 string
	InitialComponentImage      string
	InitialComponentPullPolicy string
	DefaultEntrypoint          string
	TLSMode                    string
}

type GatewayUpdateInput struct {
	Name              *string
	RestApiUrl        *string
	BaseDomain        *string
	DefaultEntrypoint *string
	TLSMode           *string
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
	Exposures   []GatewayExposureItem
}

type ProvisionGatewayInput struct {
	ProjectId      string
	InstanceKey    string
	ForceRecreate  bool
	TimeoutSeconds *int
}

type ProvisionGatewayResult struct {
	GatewayCreated   bool
	ServiceCreated   bool
	VersionPublished bool
	Ready            bool
	Gateway          GatewayView
	Version          model.Version
	Service          model.Service
	Deployment       model.Deployment
	TimedOut         bool
	Network          GatewayNetwork
	Steps            []string
}

type GatewayNetwork struct {
	Name   string
	Driver string
	Ready  bool
	Status string
}

func (i ProvisionGatewayInput) Timeout() *time.Duration {
	if i.TimeoutSeconds == nil {
		return nil
	}
	value := time.Duration(*i.TimeoutSeconds) * time.Second
	return &value
}
