package port

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// RouteConfigPublisher publishes the platform route snapshot to Traefik providers.rest.
type RouteConfigPublisher interface {
	ApplySnapshot(ctx context.Context, restApiUrl string, routes []model.Route) error
	WriteCertificate(ctx context.Context, routeName string, certPEM string, certKey string) error
	RevokeCertificate(ctx context.Context, routeName string) error
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
}

type TraefikRouterClient interface {
	ListRouters(ctx context.Context, restApiUrl string) ([]TraefikRouter, error)
	IsConnectionError(err error) bool
}
