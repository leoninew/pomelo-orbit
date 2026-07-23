package cdsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const (
	// gatewayManagedComponentName is the Version component upserted by compile.
	gatewayManagedComponentName = "traefik"
	// gatewayCompileVersionLabel is the label for the first managed unpublished version.
	gatewayCompileVersionLabel = "managed"
	// managed mount targets overwritten on each compile (Spec D3 merge-by-target).
	gatewayMountTargetDockerSock = "/var/run/docker.sock"
	gatewayMountTargetTraefikYml = "/etc/traefik/traefik.yml"
	gatewayMountTargetAcmeJSON   = "/letsencrypt/acme.json"
)

// CompileGatewayToVersion ensures an unpublished Version and upserts the managed
// traefik component (image, ports, docker.sock + traefik.yml content). Non-managed
// mounts (by target) and non-managed components are preserved.
// tcpListens are public TCP host ports to publish as Traefik entryPoints.
func (s Service) CompileGatewayToVersion(ctx context.Context, app model.Application, cfg model.GatewayConfig, tcpListens ...int) (string, error) {
	if strings.TrimSpace(app.Kind) != status.ApplicationKindGateway {
		return "", apperror.New(apperror.KindValidation, "CompileGatewayToVersion requires kind=gateway")
	}
	version, err := s.ensureUnpublishedGatewayVersion(ctx, app.Id)
	if err != nil {
		return "", err
	}
	existing, err := s.store.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to list components for compile", err)
	}
	managed, err := buildManagedGatewayComponent(cfg, existing, normalizeTCPListens(tcpListens))
	if err != nil {
		return "", apperror.New(apperror.KindValidation, err.Error())
	}
	components := upsertComponentByName(existing, managed)
	if err := validateVersionComponents(components); err != nil {
		return "", apperror.New(apperror.KindValidation, err.Error())
	}
	for i := range components {
		if components[i].Id == "" {
			components[i].Id = idutil.NewId()
		}
		components[i].VersionId = version.Id
	}
	if err := s.store.ReplaceVersionComponents(ctx, version.Id, components); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to write compiled gateway components", err)
	}
	return version.Id, nil
}

func (s Service) ensureUnpublishedGatewayVersion(ctx context.Context, applicationId string) (model.Version, error) {
	versions, err := s.store.ListVersions(ctx, applicationId)
	if err != nil {
		return model.Version{}, apperror.Wrap(apperror.KindInternal, "Failed to list versions for compile", err)
	}
	for _, v := range versions {
		if v.Status == status.VersionStatusUnpublished {
			return v, nil
		}
	}
	version := model.Version{
		Id:            idutil.NewId(),
		ApplicationId: applicationId,
		Label:         gatewayCompileVersionLabel,
		Status:        status.VersionStatusUnpublished,
	}
	if err := s.store.CreateVersion(ctx, version); err != nil {
		return model.Version{}, apperror.Wrap(apperror.KindInternal, "Failed to create managed version", err)
	}
	return version, nil
}

func normalizeTCPListens(listens []int) []int {
	seen := map[int]struct{}{}
	out := make([]int, 0, len(listens))
	for _, p := range listens {
		if p < 1 || p > 65535 {
			continue
		}
		// reserve HTTP/dashboard ports from becoming tcpN duplicates of static ones
		if p == 80 || p == 443 || p == 8080 {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	sort.Ints(out)
	return out
}

func buildManagedGatewayComponent(cfg model.GatewayConfig, existing []model.VersionComponent, tcpListens []int) (model.VersionComponent, error) {
	if cfg.Image == nil || strings.TrimSpace(*cfg.Image) == "" {
		return model.VersionComponent{}, fmt.Errorf("gateway image is required")
	}
	image := strings.TrimSpace(*cfg.Image)
	ports := []string{"80:80", "443:443", "8080:8080"}
	for _, p := range tcpListens {
		ports = append(ports, fmt.Sprintf("%d:%d", p, p))
	}
	portsRaw, err := json.Marshal(ports)
	if err != nil {
		return model.VersionComponent{}, fmt.Errorf("marshal ports: %w", err)
	}
	portsStr := string(portsRaw)
	var priorMounts []MountSpec
	for _, c := range existing {
		if c.Name == gatewayManagedComponentName {
			parsed, err := parseMountSpecs(c.MountsJSON)
			if err != nil {
				return model.VersionComponent{}, fmt.Errorf("existing managed component mounts: %w", err)
			}
			priorMounts = parsed
			break
		}
	}
	mounts := mergeManagedGatewayMounts(priorMounts, buildManagedGatewayMounts(cfg, tcpListens))
	mountsRaw, err := json.Marshal(mounts)
	if err != nil {
		return model.VersionComponent{}, fmt.Errorf("marshal mounts: %w", err)
	}
	mountsStr := string(mountsRaw)
	return model.VersionComponent{
		Name:       gatewayManagedComponentName,
		Image:      image,
		PortsJSON:  &portsStr,
		MountsJSON: &mountsStr,
	}, nil
}

func buildManagedGatewayMounts(cfg model.GatewayConfig, tcpListens []int) []MountSpec {
	yml := buildTraefikStaticConfig(cfg, tcpListens)
	return []MountSpec{
		{
			SourceType: mountSourceSpecial,
			Source:     specialDockerSock,
			Target:     gatewayMountTargetDockerSock,
			ReadOnly:   true,
		},
		{
			SourceType:  mountSourceLogical,
			Source:      "traefik.yml",
			Target:      gatewayMountTargetTraefikYml,
			Content:     yml,
			ContentMode: contentModeSync,
		},
		{
			SourceType:  mountSourceLogical,
			Source:      "acme.json",
			Target:      gatewayMountTargetAcmeJSON,
			Content:     "{}",
			ContentMode: contentModeSeed,
		},
	}
}

// managedGatewayMountTargets are rewritten on every compile; other targets are kept.
func managedGatewayMountTargets() map[string]struct{} {
	return map[string]struct{}{
		gatewayMountTargetDockerSock: {},
		gatewayMountTargetTraefikYml: {},
		gatewayMountTargetAcmeJSON:   {},
	}
}

func mergeManagedGatewayMounts(existing, managed []MountSpec) []MountSpec {
	targets := managedGatewayMountTargets()
	out := make([]MountSpec, 0, len(existing)+len(managed))
	for _, m := range existing {
		if _, isManaged := targets[m.Target]; isManaged {
			continue
		}
		out = append(out, m)
	}
	out = append(out, managed...)
	return out
}

func upsertComponentByName(existing []model.VersionComponent, managed model.VersionComponent) []model.VersionComponent {
	out := make([]model.VersionComponent, 0, len(existing)+1)
	replaced := false
	for _, c := range existing {
		if c.Name == managed.Name {
			managed.Id = c.Id
			out = append(out, managed)
			replaced = true
			continue
		}
		out = append(out, c)
	}
	if !replaced {
		out = append(out, managed)
	}
	return out
}

// buildTraefikStaticConfig generates Traefik static file with rest + docker providers and TCP entryPoints.
func buildTraefikStaticConfig(cfg model.GatewayConfig, tcpListens []int) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(`
api:
  dashboard: true
  insecure: true

entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"
`))
	b.WriteByte('\n')
	for _, p := range tcpListens {
		b.WriteString("  ")
		b.WriteString(tcpEntrypointName(p))
		b.WriteString(":\n    address: \":")
		b.WriteString(strconv.Itoa(p))
		b.WriteString("\"\n")
	}
	b.WriteString(strings.TrimSpace(`
providers:
  docker:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false
    network: traefik
  rest:
    insecure: true

log:
  level: INFO
`))
	b.WriteByte('\n')
	tlsMode := strings.ToLower(strings.TrimSpace(cfg.TLSMode))
	if tlsMode == "letsencrypt" {
		b.WriteString(strings.TrimSpace(`
certificatesResolvers:
  letsencrypt:
    acme:
      email: admin@localhost
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web
`))
		b.WriteByte('\n')
	}
	return b.String()
}

// extractCompiledTCPListens reads tcp{N} ports from the managed component ports JSON.
func extractCompiledTCPListens(components []model.VersionComponent) []int {
	for _, c := range components {
		if c.Name != gatewayManagedComponentName {
			continue
		}
		raw, err := parseAnyJSON(c.PortsJSON)
		if err != nil || raw == nil {
			return nil
		}
		items, ok := raw.([]any)
		if !ok {
			return nil
		}
		var listens []int
		for _, item := range items {
			s, ok := item.(string)
			if !ok {
				continue
			}
			// "6379:6379"
			parts := strings.Split(s, ":")
			if len(parts) != 2 {
				continue
			}
			hostPort, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				continue
			}
			if hostPort == 80 || hostPort == 443 || hostPort == 8080 {
				continue
			}
			listens = append(listens, hostPort)
		}
		return normalizeTCPListens(listens)
	}
	return nil
}

func tcpListensEqual(a, b []int) bool {
	a = normalizeTCPListens(a)
	b = normalizeTCPListens(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
