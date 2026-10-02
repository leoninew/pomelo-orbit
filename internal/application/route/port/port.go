package port

import (
	"context"
	"time"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// TransactionRunner runs a short application-owned database transaction.
// External calls must happen after the callback returns successfully.
type TransactionRunner interface {
	RunInTransaction(context.Context, func(context.Context) error) error
}

// RouteConfigPublisher publishes independently owned Route files.
type RouteConfigPublisher interface {
	// WaitUntilReady blocks until the Traefik control-plane REST API accepts
	// requests, or until ctx ends / the readiness deadline elapses. Gateway
	// deploy uses this after compose up because process start lags the container.
	WaitUntilReady(ctx context.Context, projectId string, gateway model.GatewayConfig, timeout time.Duration) error
	LockGateway(context.Context, string) (func(), error)
	InspectPublications(context.Context, string, model.GatewayConfig) ([]Publication, error)
	PublishRoute(context.Context, string, model.GatewayConfig, model.Route, string) (PublicationResult, error)
	VerifyPublished(context.Context, string, model.GatewayConfig) error
	ValidateGateway(context.Context, string, model.GatewayConfig, []model.Route) (string, error)
}

type Publication struct {
	Route                     model.Route  `json:"route"`
	Fingerprint               string       `json:"fingerprint"`
	CertificateRevision       string       `json:"certificate_revision"`
	CertificateFingerprint    string       `json:"certificate_fingerprint"`
	OperationId               string       `json:"operation_id"`
	Phase                     string       `json:"phase"`
	PreviousFingerprint       string       `json:"previous_fingerprint"`
	Previous                  *Publication `json:"previous,omitempty"`
	TargetRevision            string       `json:"target_revision"`
	GatewayApplicationId      string       `json:"gateway_application_id"`
	ActualFingerprint         string       `json:"actual_fingerprint"`
	ActualCertificateRevision string       `json:"actual_certificate_revision"`
}

type PublicationResult struct {
	OperationId             string
	FileCommit              string
	ConfigurationMatch      string
	CertificateVerification string
	Recovery                string
	Cleanup                 string
}

type RouteCertificateGenerator interface {
	Generate(ctx context.Context, domain string) (string, string, error)
}

// TraefikRouter is the router data exposed by the Traefik integration.
type TraefikRouter struct {
	Name        string
	Provider    string
	Status      string
	Rule        string
	Service     string
	Entrypoints []string
	TLS         bool
	TLSConfig   string
}

type TraefikService struct {
	Name     string
	Provider string
	Status   string
	Protocol string
	Servers  []string
}

type TraefikRouterClient interface {
	ListRouters(ctx context.Context, projectId string, gateway model.GatewayConfig) ([]TraefikRouter, error)
	ListServices(ctx context.Context, projectId string, gateway model.GatewayConfig) ([]TraefikService, error)
	// TraefikUnavailableMessage reports the safe access context for a failed
	// Traefik REST request. Other errors, including configuration and response
	// decoding failures, must return ok=false.
	TraefikUnavailableMessage(err error) (message string, ok bool)
}
