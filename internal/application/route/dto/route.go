package dto

type RouteCreateInput struct {
	Name                  string
	Protocol              string
	Domain                string
	PathPrefix            string
	TargetUrl             string
	ListenPort            *int
	ServiceId             string
	ComponentName         string
	EndpointProtocol      string
	EndpointContainerPort *int
	Enabled               bool
}

type RouteUpdateInput struct {
	Name                  *string
	Protocol              *string
	Domain                *string
	PathPrefix            *string
	TargetUrl             *string
	ListenPort            *int
	ServiceId             *string
	ComponentName         *string
	EndpointProtocol      *string
	EndpointContainerPort *int
	Enabled               *bool
}

// TraefikConfigView is the application-layer dashboard route state (not an API DTO).
type TraefikConfigView struct {
	DashboardDomain string
	HTTPSEnabled    bool
	BaseDomain      string
}
