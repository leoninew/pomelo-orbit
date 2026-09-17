package dto

import "github.com/leoninew/pomelo-orbit/internal/model"

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

// RouteDefinitionInput is a complete Route configuration for internal
// workflows. Storage and Project identity are assigned by the Route domain.
type RouteDefinitionInput struct {
	Route model.Route
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

type RouteSyncChange struct {
	RouteId string
	Enabled bool
}

type RouteSyncConfirmInput struct {
	Changes      []RouteSyncChange
	BusinessHash string
	TraefikHash  string
}

type RouteSyncDiff struct {
	Action        string
	RouteName     string
	Field         string
	BusinessValue string
	TraefikValue  string
}

type RouteSyncPreview struct {
	BusinessHash string
	TraefikHash  string
	Matched      bool
	Differences  []RouteSyncDiff
}

// TraefikConfigView is the application-layer dashboard route state (not an API DTO).
type TraefikConfigView struct {
	DashboardDomain string
	HTTPSEnabled    bool
	BaseDomain      string
}
