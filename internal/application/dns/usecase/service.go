package dnssvc

import (
	"context"
	"errors"
	"net/netip"
	"regexp"
	"strings"

	dnsdto "github.com/leoninew/pomelo-orbit/internal/application/dns/dto"
	dnsport "github.com/leoninew/pomelo-orbit/internal/application/dns/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

var resourceIdPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

type Service struct {
	project  dnsport.ProjectReader
	gateway  dnsport.GatewayReader
	provider dnsport.Provider
}

func New(project dnsport.ProjectReader, gateway dnsport.GatewayReader, provider dnsport.Provider) Service {
	return Service{project: project, gateway: gateway, provider: provider}
}

func (s Service) tokenForUser(ctx context.Context, userId, projectId string) (string, error) {
	if strings.TrimSpace(projectId) == "" {
		return "", apperror.New(apperror.KindValidation, "project_id is required")
	}
	if _, err := s.project.Project(ctx, projectId); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", apperror.New(apperror.KindNotFound, "Project not found")
		}
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.project.IsProjectMember(ctx, projectId, userId)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return "", apperror.New(apperror.KindForbidden, "Permission denied")
	}
	cfg, err := s.gateway.GatewayConfigByProject(ctx, projectId)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load gateway DNS configuration", err)
	}
	token := strings.TrimSpace(cfg.DNSApiToken)
	if errors.Is(err, repository.ErrNotFound) || token == "" {
		return "", apperror.NewWithCode(apperror.KindValidation, "dns_not_configured", "Configure a Cloudflare DNS API token in this environment's Gateway certificate settings")
	}
	return token, nil
}

func (s Service) ListZones(ctx context.Context, userId, projectId string) ([]dnsdto.Zone, error) {
	token, err := s.tokenForUser(ctx, userId, projectId)
	if err != nil {
		return nil, err
	}
	return s.provider.ListZones(ctx, token)
}

func (s Service) ListARecords(ctx context.Context, userId, projectId, zoneId, search string, page, perPage int) (repository.Page[dnsdto.Record], error) {
	token, err := s.tokenForUser(ctx, userId, projectId)
	if err != nil {
		return repository.Page[dnsdto.Record]{}, err
	}
	if !resourceIdPattern.MatchString(zoneId) {
		return repository.Page[dnsdto.Record]{}, apperror.New(apperror.KindValidation, "Invalid Cloudflare Zone Id")
	}
	if page < 1 || perPage < 1 || perPage > 100 {
		return repository.Page[dnsdto.Record]{}, apperror.New(apperror.KindValidation, "page must be positive and per_page must be between 1 and 100")
	}
	return s.provider.ListARecords(ctx, token, zoneId, strings.TrimSpace(search), page, perPage)
}

func (s Service) UpdateARecord(ctx context.Context, userId, projectId, zoneId, recordId string, input dnsdto.UpdateRecordInput) (dnsdto.Record, error) {
	token, err := s.tokenForUser(ctx, userId, projectId)
	if err != nil {
		return dnsdto.Record{}, err
	}
	if !resourceIdPattern.MatchString(zoneId) || !resourceIdPattern.MatchString(recordId) {
		return dnsdto.Record{}, apperror.New(apperror.KindValidation, "Invalid Cloudflare Zone or record Id")
	}
	input, err = validateRecordInput(input)
	if err != nil {
		return dnsdto.Record{}, err
	}
	record, err := s.provider.Record(ctx, token, zoneId, recordId)
	if err != nil {
		return dnsdto.Record{}, err
	}
	if record.Type != "A" {
		return dnsdto.Record{}, apperror.New(apperror.KindValidation, "Only existing A records can be modified")
	}
	if input.Proxied && !record.Proxiable {
		return dnsdto.Record{}, apperror.New(apperror.KindValidation, "This record does not support Cloudflare proxying")
	}
	if input.Proxied {
		input.TTL = 1
	}
	return s.provider.UpdateARecord(ctx, token, zoneId, recordId, record.Name, input)
}

func (s Service) CreateARecord(ctx context.Context, userId, projectId, zoneId string, input dnsdto.CreateRecordInput) (dnsdto.Record, error) {
	token, err := s.tokenForUser(ctx, userId, projectId)
	if err != nil {
		return dnsdto.Record{}, err
	}
	if !resourceIdPattern.MatchString(zoneId) {
		return dnsdto.Record{}, apperror.New(apperror.KindValidation, "Invalid Cloudflare Zone Id")
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 253 {
		return dnsdto.Record{}, apperror.New(apperror.KindValidation, "name is required and must not exceed 253 characters")
	}
	input.UpdateRecordInput, err = validateRecordInput(input.UpdateRecordInput)
	if err != nil {
		return dnsdto.Record{}, err
	}
	if input.Proxied {
		input.TTL = 1
	}
	return s.provider.CreateARecord(ctx, token, zoneId, input)
}

func validateRecordInput(input dnsdto.UpdateRecordInput) (dnsdto.UpdateRecordInput, error) {
	address, err := netip.ParseAddr(strings.TrimSpace(input.Content))
	if err != nil || !address.Is4() {
		return input, apperror.New(apperror.KindValidation, "content must be a valid IPv4 address")
	}
	if input.TTL != 1 && (input.TTL < 30 || input.TTL > 86400) {
		return input, apperror.New(apperror.KindValidation, "ttl must be 1 (automatic) or between 30 and 86400 seconds")
	}
	input.Content = address.String()
	return input, nil
}

func (s Service) DeleteARecord(ctx context.Context, userId, projectId, zoneId, recordId string) error {
	token, err := s.tokenForUser(ctx, userId, projectId)
	if err != nil {
		return err
	}
	if !resourceIdPattern.MatchString(zoneId) || !resourceIdPattern.MatchString(recordId) {
		return apperror.New(apperror.KindValidation, "Invalid Cloudflare Zone or record Id")
	}
	record, err := s.provider.Record(ctx, token, zoneId, recordId)
	if err != nil {
		return err
	}
	if record.Type != "A" {
		return apperror.New(apperror.KindValidation, "Only existing A records can be deleted")
	}
	return s.provider.DeleteARecord(ctx, token, zoneId, recordId)
}
