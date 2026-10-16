//go:build !unit

package routesvc

import (
	"context"
	"errors"
	"testing"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
)

func TestMkcertDoesNotSaveFailedOrStaleCertificate(t *testing.T) {
	generationFailure := errors.New("certificate generation failed")
	for _, changed := range []bool{false, true} {
		name := "generation failure"
		if changed {
			name = "route changed during generation"
		}
		t.Run(name, func(t *testing.T) {
			service, _, _, database := newRouteIntegrationService(t)
			t.Cleanup(func() { _ = database.Close() })
			ctx := context.Background()
			route, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{Name: "cert-route", Protocol: "http", Domain: "old.example.test", PathPrefix: "/", TargetUrl: "http://localhost:8090"})
			if err != nil {
				t.Fatal(err)
			}
			service.certificateGenerator = certificateGeneratorFunc(func(ctx context.Context, domain string) (string, string, error) {
				if _, ok := tx.TxFrom(ctx); ok {
					t.Fatal("generator received database transaction")
				}
				if domain != route.Domain {
					t.Fatalf("certificate domain=%q", domain)
				}
				if !changed {
					return "", "", generationFailure
				}
				updated := route
				updated.Domain = "new.example.test"
				if err := service.route.UpdateRoute(ctx, routeTestProjectId, updated.Route); err != nil {
					t.Fatal(err)
				}
				return "certificate", "private-key", nil
			})
			_, err = service.EnableRouteMkcert(ctx, routeTestUserId, routeTestProjectId, route.Id)
			if changed && !apperror.IsKind(err, apperror.KindConflict) {
				t.Fatalf("expected stale certificate conflict, got %v", err)
			}
			if !changed && !errors.Is(err, generationFailure) {
				t.Fatalf("expected generation failure, got %v", err)
			}
			stored, err := service.route.Route(ctx, routeTestProjectId, route.Id)
			if err != nil {
				t.Fatal(err)
			}
			if stored.CertPEM != nil || stored.CertKey != nil || stored.HTTPSEnabled {
				t.Fatalf("failed generation saved certificate: %+v", stored)
			}
			if changed && stored.Domain != "new.example.test" {
				t.Fatalf("route change overwritten: %+v", stored)
			}
		})
	}
}

type certificateGeneratorFunc func(context.Context, string) (string, string, error)

func (f certificateGeneratorFunc) Generate(ctx context.Context, domain string) (string, string, error) {
	return f(ctx, domain)
}
