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

// RouteConfigPublisher publishes the platform route snapshot to Traefik providers.rest.
type RouteConfigPublisher interface {
	// WaitUntilReady blocks until the Traefik control-plane REST API accepts
	// requests, or until ctx ends / the readiness deadline elapses. Gateway
	// deploy uses this after compose up because process start lags the container.
	WaitUntilReady(ctx context.Context, restApiUrl string, timeout time.Duration) error
	ApplySnapshot(ctx context.Context, gateway model.GatewayConfig, routes []model.Route) error
}

// SnapshotPublisher republishes the full enabled Route configuration after a
// Gateway runtime has been recreated.
type SnapshotPublisher interface {
	PublishSnapshot(ctx context.Context) error
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
	ListRouters(ctx context.Context, restApiUrl string) ([]TraefikRouter, error)
	ListServices(ctx context.Context, restApiUrl string) ([]TraefikService, error)
	IsConnectionError(err error) bool
}
