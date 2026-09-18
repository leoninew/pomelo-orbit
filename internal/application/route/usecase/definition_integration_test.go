package routesvc

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
	applicationrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/application"
	gatewayrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/gateway"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
	routerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/route"
	servicerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/service"
)

func TestCreateRouteFromDefinitionPreservesConfigurationAndDisablesRoutes(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := database.ExecContext(ctx, `INSERT INTO project (id, name, code) VALUES ('project-1', 'Project', 'project')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO project_member (project_id, user_id) VALUES ('project-1', 'user-1')`); err != nil {
		t.Fatal(err)
	}
	service := New(
		projectrepo.NewRepository(database),
		applicationrepo.NewRepository(database),
		servicerepo.NewRepository(database),
		routerepo.NewRepository(database),
		gatewayrepo.NewRepository(database),
		nil,
		nil,
		nil,
		nil,
	)
	certPEM, certKey := "-----BEGIN CERTIFICATE-----\nsource\n-----END CERTIFICATE-----", "-----BEGIN PRIVATE KEY-----\nsource\n-----END PRIVATE KEY-----"
	created, err := service.CreateRouteFromDefinition(ctx, "user-1", "project-1", routedto.RouteDefinitionInput{Route: model.Route{
		Id: "source-route", Name: "api", Protocol: routeProtocolHTTP, Domain: "api.example.test", PathPrefix: "/v1",
		TargetUrl: "http://upstream.internal:8080", Enabled: true, HTTPSEnabled: true,
		CertPEM: &certPEM, CertKey: &certKey, CertType: certTypeManual, AcmeChallenge: acmeChallengeHTTP,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if created.Id == "source-route" || created.ProjectId == nil || *created.ProjectId != "project-1" {
		t.Fatalf("created route identity = %+v", created)
	}
	if created.Enabled {
		t.Fatalf("created route remained enabled: %+v", created)
	}
	if !created.HTTPSEnabled || created.CertPEM == nil || *created.CertPEM != certPEM || created.CertKey == nil || *created.CertKey != certKey {
		t.Fatalf("created route certificate = %+v", created)
	}
	if created.TargetUrl != "http://upstream.internal:8080" || created.PathPrefix != "/v1" {
		t.Fatalf("created route configuration = %+v", created)
	}

	serviceID, componentName, protocol := "source-service", "app", "http"
	containerPort := 8080
	stale, err := service.CreateRouteFromDefinition(ctx, "user-1", "project-1", routedto.RouteDefinitionInput{Route: model.Route{
		Id: "source-stale-route", Name: "stale", Protocol: routeProtocolHTTP, Domain: "stale.example.test", PathPrefix: "/",
		TargetUrl: "source-service/app/http8080", ServiceId: &serviceID, ComponentName: &componentName,
		EndpointProtocol: &protocol, EndpointContainerPort: &containerPort, Enabled: false,
		CertType: certTypeManual, AcmeChallenge: acmeChallengeHTTP,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if stale.Enabled || stale.ServiceId == nil || *stale.ServiceId != serviceID || stale.ComponentName == nil || *stale.ComponentName != componentName || stale.EndpointProtocol == nil || *stale.EndpointProtocol != protocol || stale.EndpointContainerPort == nil || *stale.EndpointContainerPort != containerPort {
		t.Fatalf("restored disabled managed route = %+v", stale)
	}
}
