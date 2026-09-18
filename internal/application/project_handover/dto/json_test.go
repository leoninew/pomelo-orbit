package dto

import (
	"strings"
	"testing"

	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	projectdto "github.com/leoninew/pomelo-orbit/internal/application/project/dto"
	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestEncodeDecodeKeepsInternalReferencesWithoutProjectOwnership(t *testing.T) {
	projectID := "source-project"
	serviceID := "source-service"
	item := Package{
		Format:  Format,
		Version: FormatVersion,
		Project: projectdto.ProjectDefinition{Name: "Source", Code: "source", IsActive: true},
		Services: []servicedto.ServiceDefinition{{
			Service: model.Service{
				Id: "source-service", ProjectId: projectID, ApplicationId: "source-application",
				VersionId: "source-version", Code: "web", Status: "running",
			},
			Components: []model.ServiceComponent{{
				Id: "source-service-component", ServiceId: serviceID,
				SourceVersionComponentId: "source-version-component", ComponentName: "web",
			}},
		}},
		Routes: []routedto.RouteDefinitionInput{{Route: model.Route{
			Id: "source-route", ProjectId: &projectID, ServiceId: &serviceID, Name: "web",
		}}},
	}

	document, err := Encode(item)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(document), "project_id") || strings.Contains(string(document), `"id":"source-project"`) {
		t.Fatalf("handover document retained source Project identity: %s", document)
	}

	decoded, err := Decode(document)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Services[0].Service.Id != "source-service" || decoded.Services[0].Service.ProjectId != "" {
		t.Fatalf("unexpected decoded Service: %+v", decoded.Services[0].Service)
	}
	if decoded.Services[0].Components[0].SourceVersionComponentId != "source-version-component" {
		t.Fatalf("source Component reference was not preserved: %+v", decoded.Services[0].Components[0])
	}
	if decoded.Routes[0].Route.ServiceId == nil || *decoded.Routes[0].Route.ServiceId != serviceID || decoded.Routes[0].Route.ProjectId != nil {
		t.Fatalf("unexpected decoded Route: %+v", decoded.Routes[0].Route)
	}
}

func TestDecodeRejectsProjectOwnershipAndUnknownFields(t *testing.T) {
	for _, document := range []string{
		`{"format":"pomelo-orbit/project-handover","version":1,"project":{"id":"source","name":"Source","code":"source","is_active":true}}`,
		`{"format":"pomelo-orbit/project-handover","version":1,"project":{"name":"Source","code":"source","is_active":true},"services":[{"service":{"project_id":"source"}}]}`,
		`{"format":"pomelo-orbit/project-handover","version":1,"project":{"name":"Source","code":"source","is_active":true},"unexpected":true}`,
	} {
		if _, err := Decode([]byte(document)); err == nil {
			t.Fatalf("expected Decode to reject %s", document)
		}
	}
}

func TestEncodeDecodeKeepsGatewayNetworkName(t *testing.T) {
	document, err := Encode(Package{
		Format:  Format,
		Version: FormatVersion,
		Gateway: &gatewaydto.GatewayDefinition{Config: model.GatewayConfig{
			NetworkName: "traefik",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(document), `"network_name":"traefik"`) {
		t.Fatalf("handover document did not use network_name: %s", document)
	}

	decoded, err := Decode(document)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Gateway == nil || decoded.Gateway.Config.NetworkName != "traefik" {
		t.Fatalf("gateway network name = %+v, want traefik", decoded.Gateway)
	}
}
