package model

// EffectiveServicePlan is the only runtime configuration consumed by preview
// and deployment. It combines a Version declaration with one Service's sparse
// component overlays and, for gateways, a platform policy.
type EffectiveServicePlan struct {
	Application Application
	Version     Version
	Service     Service
	Components  []EffectiveServiceComponent
	Gateway     *GatewayConfig
	// JoinTraefikNetwork controls the shared platform network projection for a
	// standard Service. Nil retains the product default of joining it.
	JoinTraefikNetwork *bool
}

func (p EffectiveServicePlan) JoinsTraefikNetwork() bool {
	return p.JoinTraefikNetwork == nil || *p.JoinTraefikNetwork
}

type EffectiveServiceComponent struct {
	ServiceComponentId string
	SourceComponentId  string
	Name               string
	Image              string
	Entrypoint         []string
	Command            []string
	Env                []VersionComponentEnv
	Mounts             []VersionComponentMount
	Dependencies       []VersionComponentDependency
	Healthcheck        *VersionComponentHealthcheck
	Resources          *VersionComponentResources
	PullPolicy         string
	RestartPolicy      *string
	Tmpfs              []VersionComponentTmpfs
	Ulimits            []VersionComponentUlimit
	Devices            []VersionComponentDeviceRequest
	Endpoints          []VersionComponentEndpoint
}
