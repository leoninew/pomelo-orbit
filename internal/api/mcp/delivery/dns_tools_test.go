package delivery

import (
	"context"
	"testing"

	dnsdto "github.com/leoninew/pomelo-orbit/internal/application/dns/dto"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type dnsToolService struct {
	userId, projectId, zoneId, recordId string
	input                               dnsdto.UpdateRecordInput
}

func (s *dnsToolService) ListZones(_ context.Context, userId, projectId string) ([]dnsdto.Zone, error) {
	s.userId, s.projectId = userId, projectId
	return []dnsdto.Zone{{Id: "zone", Name: "example.test"}}, nil
}
func (s *dnsToolService) ListARecords(_ context.Context, userId, projectId, zoneId, search string, page, perPage int) (repository.Page[dnsdto.Record], error) {
	s.userId, s.projectId, s.zoneId = userId, projectId, zoneId
	return repository.Page[dnsdto.Record]{Items: []dnsdto.Record{{Id: "record", Type: "A", Content: "192.0.2.1"}}, Page: page, PerPage: perPage, Total: 1}, nil
}
func (s *dnsToolService) CreateARecord(_ context.Context, userId, projectId, zoneId string, input dnsdto.CreateRecordInput) (dnsdto.Record, error) {
	s.userId, s.projectId, s.zoneId = userId, projectId, zoneId
	return dnsdto.Record{Id: "record", Name: input.Name}, nil
}
func (s *dnsToolService) DeleteARecord(_ context.Context, userId, projectId, zoneId, recordId string) error {
	s.userId, s.projectId, s.zoneId, s.recordId = userId, projectId, zoneId, recordId
	return nil
}
func (s *dnsToolService) UpdateARecord(_ context.Context, userId, projectId, zoneId, recordId string, input dnsdto.UpdateRecordInput) (dnsdto.Record, error) {
	s.userId, s.projectId, s.zoneId, s.recordId, s.input = userId, projectId, zoneId, recordId, input
	return dnsdto.Record{Id: recordId, Type: "A", Content: input.Content, TTL: input.TTL, Proxied: input.Proxied}, nil
}

func TestDNSToolsUseSelectedProjectWithoutRuntimeOrTransaction(t *testing.T) {
	service := &dnsToolService{}
	// Environment, Gateway runtime and TransactionRunner deliberately absent.
	server, err := NewServer(Dependencies{ActorUserId: "actor", SelectedProjectId: "project-1", DNS: service})
	if err != nil {
		t.Fatal(err)
	}
	session := connectInMemory(t, server)
	for _, tc := range []struct {
		name      string
		arguments map[string]any
	}{
		{"orbit_list_dns_zones", map[string]any{}},
		{"orbit_list_dns_a_records", map[string]any{"zone_id": "zone", "page": 2, "per_page": 10}},
		{"orbit_create_dns_a_record", map[string]any{"zone_id": "zone", "name": "api.example.test", "content": "192.0.2.2", "ttl": 1, "proxied": false}},
		{"orbit_delete_dns_a_record", map[string]any{"zone_id": "zone", "record_id": "record"}},
		{"orbit_update_dns_a_record", map[string]any{"zone_id": "zone", "record_id": "record", "content": "192.0.2.2", "ttl": 300, "proxied": false}},
	} {
		result, callErr := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tc.name, Arguments: tc.arguments})
		if callErr != nil {
			t.Fatal(callErr)
		}
		if result.IsError {
			t.Fatalf("%s failed: %#v", tc.name, result.Content)
		}
		if service.userId != "actor" || service.projectId != "project-1" {
			t.Fatalf("scope=%q/%q", service.userId, service.projectId)
		}
		if tc.name == "orbit_update_dns_a_record" {
			output := structuredOutput(t, result)
			if output["record"].(map[string]any)["content"] != "192.0.2.2" || service.zoneId != "zone" || service.recordId != "record" || service.input.Content != "192.0.2.2" || service.input.TTL != 300 || service.input.Proxied {
				t.Fatalf("output=%#v input=%#v", output, service.input)
			}
		}
	}
}
