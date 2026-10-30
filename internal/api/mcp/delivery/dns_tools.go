package delivery

import (
	"context"

	dnsdto "github.com/leoninew/pomelo-orbit/internal/application/dns/dto"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (c *core) registerDNSTools(server *mcp.Server) {
	addTool(server, "orbit_list_dns_zones", "List Cloudflare Zones accessible to the selected Project environment's Gateway DNS token. Requires Zone Read permission; no running Gateway is needed.", func(ctx context.Context, _ struct{}) (map[string]any, error) {
		projectId, err := c.currentProjectId()
		if err != nil {
			return nil, err
		}
		zones, err := c.deps.DNS.ListZones(ctx, c.deps.ActorUserId, projectId)
		if err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(zones))
		for _, zone := range zones {
			items = append(items, map[string]any{"id": zone.Id, "name": zone.Name})
		}
		return map[string]any{"zones": items}, nil
	})
	addTool(server, "orbit_list_dns_a_records", "List only existing A records in a Cloudflare Zone using the selected Project's Gateway DNS token. Read the current values before updating a record.", func(ctx context.Context, input struct {
		ZoneId  string `json:"zone_id" jsonschema:"required"`
		Search  string `json:"search,omitempty"`
		Page    int    `json:"page,omitempty"`
		PerPage int    `json:"per_page,omitempty"`
	}) (map[string]any, error) {
		projectId, err := c.currentProjectId()
		if err != nil {
			return nil, err
		}
		if input.Page == 0 {
			input.Page = 1
		}
		if input.PerPage == 0 {
			input.PerPage = 20
		}
		result, err := c.deps.DNS.ListARecords(ctx, c.deps.ActorUserId, projectId, input.ZoneId, input.Search, input.Page, input.PerPage)
		if err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(result.Items))
		for _, record := range result.Items {
			items = append(items, dnsRecordOutput(record))
		}
		return map[string]any{"records": items, "total": result.Total, "page": result.Page, "per_page": result.PerPage}, nil
	})
	addTool(server, "orbit_create_dns_a_record", "Create a Cloudflare A record in the selected Project's Zone. This writes external DNS immediately; do not retry blindly after a timeout.", func(ctx context.Context, input struct {
		ZoneId  string `json:"zone_id" jsonschema:"required"`
		Name    string `json:"name" jsonschema:"required"`
		Content string `json:"content" jsonschema:"required"`
		TTL     int    `json:"ttl" jsonschema:"required"`
		Proxied bool   `json:"proxied" jsonschema:"required"`
	}) (map[string]any, error) {
		projectId, err := c.currentProjectId()
		if err != nil {
			return nil, err
		}
		record, err := c.deps.DNS.CreateARecord(ctx, c.deps.ActorUserId, projectId, input.ZoneId, dnsdto.CreateRecordInput{Name: input.Name, UpdateRecordInput: dnsdto.UpdateRecordInput{Content: input.Content, TTL: input.TTL, Proxied: input.Proxied}})
		if err != nil {
			return nil, err
		}
		return writeResult("create_dns_a_record", map[string]string{"zone_id": input.ZoneId, "record_id": record.Id}, "POST", "/api/dns/zone/"+input.ZoneId+"/record", map[string]any{"record": dnsRecordOutput(record)}), nil
	})
	addTool(server, "orbit_delete_dns_a_record", "Delete an existing Cloudflare A record. Confirm the target with the user before deletion. This immediately changes external DNS; do not retry blindly after a timeout.", func(ctx context.Context, input struct {
		ZoneId   string `json:"zone_id" jsonschema:"required"`
		RecordId string `json:"record_id" jsonschema:"required"`
	}) (map[string]any, error) {
		projectId, err := c.currentProjectId()
		if err != nil {
			return nil, err
		}
		if err := c.deps.DNS.DeleteARecord(ctx, c.deps.ActorUserId, projectId, input.ZoneId, input.RecordId); err != nil {
			return nil, err
		}
		return writeResult("delete_dns_a_record", map[string]string{"zone_id": input.ZoneId, "record_id": input.RecordId}, "DELETE", "/api/dns/zone/"+input.ZoneId+"/record/"+input.RecordId, nil), nil
	})
	addTool(server, "orbit_update_dns_a_record", "Modify an existing Cloudflare A record's IPv4 content, TTL and proxy status after reviewing its current values. Requires DNS Edit permission. Keeps name and other fields unchanged. ttl=1 means automatic; enabling proxy forces automatic TTL. This writes external DNS immediately; do not retry blindly after a timeout.", func(ctx context.Context, input struct {
		ZoneId   string `json:"zone_id" jsonschema:"required"`
		RecordId string `json:"record_id" jsonschema:"required"`
		Content  string `json:"content" jsonschema:"required"`
		TTL      int    `json:"ttl" jsonschema:"required"`
		Proxied  bool   `json:"proxied" jsonschema:"required"`
	}) (map[string]any, error) {
		projectId, err := c.currentProjectId()
		if err != nil {
			return nil, err
		}
		record, err := c.deps.DNS.UpdateARecord(ctx, c.deps.ActorUserId, projectId, input.ZoneId, input.RecordId, dnsdto.UpdateRecordInput{Content: input.Content, TTL: input.TTL, Proxied: input.Proxied})
		if err != nil {
			return nil, err
		}
		return writeResult("update_dns_a_record", map[string]string{"zone_id": input.ZoneId, "record_id": input.RecordId}, "PUT", "/api/dns/zone/"+input.ZoneId+"/record/"+input.RecordId, map[string]any{"record": dnsRecordOutput(record)}), nil
	})
}

func dnsRecordOutput(record dnsdto.Record) map[string]any {
	return map[string]any{"id": record.Id, "name": record.Name, "type": record.Type, "content": record.Content, "ttl": record.TTL, "proxied": record.Proxied, "proxiable": record.Proxiable}
}
