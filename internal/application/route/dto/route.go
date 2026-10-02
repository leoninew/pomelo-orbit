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
	Enabled *bool
}

type RouteSyncConfirmInput struct {
	RouteIds        []string
	Changes         []RouteSyncChange
	BusinessHash    string
	PublicationHash string
}

type RouteSyncPreviewInput struct {
	Scope    string
	RouteIds []string
	Changes  []RouteSyncChange
}

type RouteSyncResult struct {
	RouteId                 string `json:"route_id"`
	RouteName               string `json:"route_name"`
	OperationId             string `json:"operation_id"`
	Code                    string `json:"code"`
	Error                   string `json:"error"`
	BusinessSave            string `json:"business_save"`
	FileCommit              string `json:"file_commit"`
	ConfigurationMatch      string `json:"configuration_match"`
	CertificateVerification string `json:"certificate_verification"`
	Recovery                string `json:"recovery"`
	Cleanup                 string `json:"cleanup"`
}

type RouteSyncConfirmResult struct {
	Code    string            `json:"code"`
	Results []RouteSyncResult `json:"results"`
}

type RouteSyncPlanItem struct {
	RouteId         string         `json:"route_id"`
	RouteName       string         `json:"route_name"`
	Action          string         `json:"action"`
	Rule            *RouteSyncRule `json:"rule"`
	CertType        string         `json:"cert_type"`
	AcmeChallenge   string         `json:"acme_challenge"`
	BusinessHash    string         `json:"business_hash"`
	PublicationHash string         `json:"publication_hash"`
}

type RouteSyncRule struct {
	Protocol string `json:"protocol"`
	Match    string `json:"match"`
	Target   string `json:"target"`
}

type RouteSyncPreview struct {
	RouteIds        []string
	PublicationHash string
	BusinessHash    string
	Items           []RouteSyncPlanItem
}

// TraefikConfigView is the application-layer dashboard route state (not an API DTO).
type TraefikConfigView struct {
	DashboardDomain string
	HTTPSEnabled    bool
	InternalDomain  string
	ExternalDomain  string
}
