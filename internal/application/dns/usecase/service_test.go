package dnssvc

import (
	"context"
	"testing"

	dnsdto "github.com/leoninew/pomelo-orbit/internal/application/dns/dto"
	dnsport "github.com/leoninew/pomelo-orbit/internal/application/dns/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

const testZoneId = "11111111111111111111111111111111"
const testRecordId = "22222222222222222222222222222222"

type projectStub struct{ member bool }

func (p projectStub) Project(context.Context, string) (model.Project, error) {
	return model.Project{Id: "project"}, nil
}
func (p projectStub) IsProjectMember(context.Context, string, string) (bool, error) {
	return p.member, nil
}

type gatewayStub struct {
	calls int
	token string
}

func (g *gatewayStub) GatewayConfigByProject(_ context.Context, projectId string) (model.GatewayConfig, error) {
	g.calls++
	return model.GatewayConfig{ApplicationId: projectId, DNSApiToken: g.token}, nil
}

type providerStub struct {
	dnsport.Provider
	calls    int
	token    string
	zoneId   string
	recordId string
	current  dnsdto.Record
	input    dnsdto.UpdateRecordInput
	readErr  error
}

func (p *providerStub) ListZones(_ context.Context, token string) ([]dnsdto.Zone, error) {
	p.calls++
	p.token = token
	return []dnsdto.Zone{{Id: testZoneId, Name: "example.test"}}, nil
}
func (p *providerStub) ListARecords(_ context.Context, token, zoneId, search string, page, perPage int) (repository.Page[dnsdto.Record], error) {
	p.calls++
	p.token = token
	p.zoneId = zoneId
	return repository.Page[dnsdto.Record]{Items: []dnsdto.Record{p.current}, Page: page, PerPage: perPage, Total: 1}, nil
}
func (p *providerStub) Record(_ context.Context, token, zoneId, recordId string) (dnsdto.Record, error) {
	p.calls++
	p.token = token
	p.zoneId = zoneId
	p.recordId = recordId
	return p.current, p.readErr
}
func (p *providerStub) UpdateARecord(_ context.Context, token, zoneId, recordId, name string, input dnsdto.UpdateRecordInput) (dnsdto.Record, error) {
	p.calls++
	p.token = token
	p.zoneId = zoneId
	p.recordId = recordId
	p.input = input
	return dnsdto.Record{Id: recordId, Name: name, Type: "A", Content: input.Content, TTL: input.TTL, Proxied: input.Proxied, Proxiable: p.current.Proxiable}, nil
}

func TestDNSRequiresProjectMembershipBeforeReadingToken(t *testing.T) {
	gateway := &gatewayStub{token: "secret"}
	provider := &providerStub{}
	svc := New(projectStub{}, gateway, provider)
	_, err := svc.ListZones(context.Background(), "outsider", "project")
	if !apperror.IsKind(err, apperror.KindForbidden) || gateway.calls != 0 || provider.calls != 0 {
		t.Fatalf("err=%v gateway calls=%d provider calls=%d", err, gateway.calls, provider.calls)
	}
}

func TestDNSReportsMissingToken(t *testing.T) {
	provider := &providerStub{}
	svc := New(projectStub{member: true}, &gatewayStub{}, provider)
	_, err := svc.ListZones(context.Background(), "actor", "project")
	if got := apperror.Classify(err); got.Code != "dns_not_configured" || provider.calls != 0 {
		t.Fatalf("error=%#v provider calls=%d", got, provider.calls)
	}
}

func TestDNSListsZonesAndARecordsWithEnvironmentToken(t *testing.T) {
	provider := &providerStub{current: dnsdto.Record{Id: testRecordId, Type: "A", Content: "192.0.2.1"}}
	svc := New(projectStub{member: true}, &gatewayStub{token: " environment-token "}, provider)
	zones, err := svc.ListZones(context.Background(), "actor", "project")
	if err != nil || len(zones) != 1 || zones[0].Id != testZoneId {
		t.Fatalf("zones=%#v err=%v", zones, err)
	}
	records, err := svc.ListARecords(context.Background(), "actor", "project", testZoneId, "", 2, 10)
	if err != nil || records.Page != 2 || records.PerPage != 10 || records.Total != 1 || len(records.Items) != 1 || provider.token != "environment-token" || provider.zoneId != testZoneId {
		t.Fatalf("records=%#v token=%q zone=%q err=%v", records, provider.token, provider.zoneId, err)
	}
}

func TestDNSRejectsInvalidRecordChangesBeforeProviderCall(t *testing.T) {
	for _, tc := range []struct {
		name, content string
		ttl           int
	}{
		{"IPv6", "2001:db8::1", 300}, {"invalid IPv4", "192.0.2.999", 300}, {"leading zero", "192.000.2.1", 300}, {"missing TTL", "192.0.2.1", 0}, {"TTL too short", "192.0.2.1", 29}, {"TTL too long", "192.0.2.1", 86401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := &providerStub{}
			svc := New(projectStub{member: true}, &gatewayStub{token: "secret"}, provider)
			_, err := svc.UpdateARecord(context.Background(), "actor", "project", testZoneId, testRecordId, dnsdto.UpdateRecordInput{Content: tc.content, TTL: tc.ttl})
			if !apperror.IsKind(err, apperror.KindValidation) || provider.calls != 0 {
				t.Fatalf("err=%v calls=%d", err, provider.calls)
			}
		})
	}
}

func TestDNSOnlyUpdatesExistingARecords(t *testing.T) {
	for _, recordType := range []string{"AAAA", "CNAME", "TXT"} {
		t.Run(recordType, func(t *testing.T) {
			provider := &providerStub{current: dnsdto.Record{Type: recordType}}
			svc := New(projectStub{member: true}, &gatewayStub{token: "secret"}, provider)
			_, err := svc.UpdateARecord(context.Background(), "actor", "project", testZoneId, testRecordId, dnsdto.UpdateRecordInput{Content: "192.0.2.1", TTL: 300})
			if !apperror.IsKind(err, apperror.KindValidation) || provider.calls != 1 {
				t.Fatalf("err=%v calls=%d", err, provider.calls)
			}
		})
	}
}

func TestDNSUpdatesAddressAndNormalizesProxiedTTL(t *testing.T) {
	provider := &providerStub{current: dnsdto.Record{Id: testRecordId, Name: "api.example.test", Type: "A", Proxiable: true}}
	svc := New(projectStub{member: true}, &gatewayStub{token: "secret"}, provider)
	result, err := svc.UpdateARecord(context.Background(), "actor", "project", testZoneId, testRecordId, dnsdto.UpdateRecordInput{Content: " 192.0.2.2 ", TTL: 300, Proxied: true})
	if err != nil || result.Content != "192.0.2.2" || result.Name != "api.example.test" || result.TTL != 1 || !result.Proxied || provider.calls != 2 || provider.recordId != testRecordId || provider.input.TTL != 1 {
		t.Fatalf("record=%#v input=%#v calls=%d err=%v", result, provider.input, provider.calls, err)
	}
	provider.readErr = apperror.New(apperror.KindNotFound, "record gone")
	provider.calls = 0
	_, err = svc.UpdateARecord(context.Background(), "actor", "project", testZoneId, testRecordId, dnsdto.UpdateRecordInput{Content: "192.0.2.2", TTL: 1})
	if !apperror.IsKind(err, apperror.KindNotFound) || provider.calls != 1 {
		t.Fatalf("err=%v calls=%d", err, provider.calls)
	}
}

func (p *providerStub) CreateARecord(_ context.Context, token, zoneId string, input dnsdto.CreateRecordInput) (dnsdto.Record, error) {
	p.calls++
	p.token = token
	p.zoneId = zoneId
	p.input = input.UpdateRecordInput
	return dnsdto.Record{Name: input.Name, Content: input.Content, TTL: input.TTL, Proxied: input.Proxied}, nil
}
func TestCreateARecordAuthorizationAndValidation(t *testing.T) {
	provider := &providerStub{}
	input := dnsdto.CreateRecordInput{Name: " api.example.test ", UpdateRecordInput: dnsdto.UpdateRecordInput{Content: " 192.0.2.1 ", TTL: 300, Proxied: true}}
	denied := New(projectStub{}, &gatewayStub{token: "secret"}, provider)
	if _, err := denied.CreateARecord(context.Background(), "outsider", "project", testZoneId, input); !apperror.IsKind(err, apperror.KindForbidden) || provider.calls != 0 {
		t.Fatalf("err=%v calls=%d", err, provider.calls)
	}
	svc := New(projectStub{member: true}, &gatewayStub{token: "secret"}, provider)
	invalid := input
	invalid.Name = "  "
	if _, err := svc.CreateARecord(context.Background(), "actor", "project", testZoneId, invalid); !apperror.IsKind(err, apperror.KindValidation) || provider.calls != 0 {
		t.Fatalf("err=%v calls=%d", err, provider.calls)
	}
	invalid = input
	invalid.Content = "invalid"
	if _, err := svc.CreateARecord(context.Background(), "actor", "project", testZoneId, invalid); !apperror.IsKind(err, apperror.KindValidation) || provider.calls != 0 {
		t.Fatalf("err=%v calls=%d", err, provider.calls)
	}
	result, err := svc.CreateARecord(context.Background(), "actor", "project", testZoneId, input)
	if err != nil || result.Name != "api.example.test" || result.Content != "192.0.2.1" || result.TTL != 1 || !result.Proxied || provider.calls != 1 {
		t.Fatalf("result=%#v err=%v calls=%d", result, err, provider.calls)
	}
}

func (p *providerStub) DeleteARecord(_ context.Context, token, zoneId, recordId string) error {
	p.calls++
	p.token = token
	p.zoneId = zoneId
	p.recordId = recordId
	return nil
}
func TestDeleteARecordChecksMembershipAndType(t *testing.T) {
	provider := &providerStub{current: dnsdto.Record{Type: "A"}}
	denied := New(projectStub{}, &gatewayStub{token: "secret"}, provider)
	if err := denied.DeleteARecord(context.Background(), "outsider", "project", testZoneId, testRecordId); !apperror.IsKind(err, apperror.KindForbidden) || provider.calls != 0 {
		t.Fatalf("err=%v calls=%d", err, provider.calls)
	}
	svc := New(projectStub{member: true}, &gatewayStub{token: "secret"}, provider)
	provider.current.Type = "CNAME"
	if err := svc.DeleteARecord(context.Background(), "actor", "project", testZoneId, testRecordId); !apperror.IsKind(err, apperror.KindValidation) || provider.calls != 1 {
		t.Fatalf("err=%v calls=%d", err, provider.calls)
	}
	provider.current.Type = "A"
	provider.calls = 0
	if err := svc.DeleteARecord(context.Background(), "actor", "project", testZoneId, testRecordId); err != nil || provider.calls != 2 || provider.recordId != testRecordId {
		t.Fatalf("err=%v calls=%d", err, provider.calls)
	}
}
