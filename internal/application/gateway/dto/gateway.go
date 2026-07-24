package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

type GatewayCreateInput struct {
	ProjectId         string
	Code              string
	Name              string
	RestApiUrl        string
	BaseDomain        string
	Image             *string
	ImagePullPolicy   string
	DefaultEntrypoint *string
	TLSMode           *string
}

type GatewayUpdateInput struct {
	Name              *string
	RestApiUrl        *string
	BaseDomain        *string
	Image             *string
	ImagePullPolicy   *string
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
