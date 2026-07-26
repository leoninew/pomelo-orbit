package gatewaysvc

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const (
	gatewayManagedComponentName  = "traefik"
	gatewayCompileVersionLabel   = "managed"
	gatewayMountTargetDockerSock = "/var/run/docker.sock"
	gatewayMountTargetTraefikYml = "/etc/traefik/traefik.yml"
	gatewayMountTargetAcmeJSON   = "/letsencrypt/acme.json"
)

// CompileGatewayToVersion ensures an unpublished version contains the managed Traefik component.
func (s Service) CompileGatewayToVersion(ctx context.Context, app model.Application, cfg model.GatewayConfig, tcpListens ...int) (string, error) {
	if strings.TrimSpace(app.Kind) != status.ApplicationKindGateway {
		return "", apperror.New(apperror.KindValidation, "CompileGatewayToVersion requires kind=gateway")
	}
	version, err := s.ensureUnpublishedGatewayVersion(ctx, app.Id)
	if err != nil {
		return "", err
	}
	existing, err := s.application.VersionComponentsByVersion(ctx, version.Id)
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
	for index := range components {
		if components[index].Id == "" {
			components[index].Id = idutil.NewId()
		}
		components[index].VersionId = version.Id
	}
	if err := s.application.ReplaceVersionComponents(ctx, version.Id, components); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to write compiled gateway components", err)
	}
	return version.Id, nil
}

func (s Service) ensureUnpublishedGatewayVersion(ctx context.Context, applicationId string) (model.Version, error) {
	versions, err := s.application.ListVersions(ctx, applicationId)
	if err != nil {
		return model.Version{}, apperror.Wrap(apperror.KindInternal, "Failed to list versions for compile", err)
	}
	for _, version := range versions {
		if version.Status == status.VersionStatusUnpublished {
			return version, nil
		}
	}
	version := model.Version{
		Id:            idutil.NewId(),
		ApplicationId: applicationId,
		Label:         gatewayCompileVersionLabel + "-" + idutil.NewId(),
		Status:        status.VersionStatusUnpublished,
	}
	if err := s.application.CreateVersion(ctx, version); err != nil {
		return model.Version{}, apperror.Wrap(apperror.KindInternal, "Failed to create managed version", err)
	}
	return version, nil
}

func buildManagedGatewayComponent(cfg model.GatewayConfig, existing []model.VersionComponent, tcpListens []int) (model.VersionComponent, error) {
	if cfg.Image == nil || strings.TrimSpace(*cfg.Image) == "" {
		return model.VersionComponent{}, fmt.Errorf("gateway image is required")
	}
	ports := []string{"80:80", "443:443", "8080:8080"}
	for _, listen := range tcpListens {
		ports = append(ports, fmt.Sprintf("%d:%d", listen, listen))
	}
	portsJSON, err := json.Marshal(ports)
	if err != nil {
		return model.VersionComponent{}, fmt.Errorf("marshal ports: %w", err)
	}
	portsRaw := string(portsJSON)
	var previousMounts []mountSpec
	for _, component := range existing {
		if component.Name != gatewayManagedComponentName {
			continue
		}
		parsed, err := parseMountSpecs(component.MountsJSON)
		if err != nil {
			return model.VersionComponent{}, fmt.Errorf("existing managed component mounts: %w", err)
		}
		previousMounts = parsed
		break
	}
	mounts := mergeManagedGatewayMounts(previousMounts, buildManagedGatewayMounts(cfg, tcpListens))
	mountsJSON, err := json.Marshal(mounts)
	if err != nil {
		return model.VersionComponent{}, fmt.Errorf("marshal mounts: %w", err)
	}
	mountsRaw := string(mountsJSON)
	return model.VersionComponent{
		Name: gatewayManagedComponentName, Image: strings.TrimSpace(*cfg.Image),
		PortsJSON: &portsRaw, MountsJSON: &mountsRaw,
	}, nil
}

func buildManagedGatewayMounts(cfg model.GatewayConfig, tcpListens []int) []mountSpec {
	return []mountSpec{
		{SourceType: mountSourceSpecial, Source: specialDockerSock, Target: gatewayMountTargetDockerSock, ReadOnly: true},
		{SourceType: mountSourceLogical, Source: "traefik.yml", Target: gatewayMountTargetTraefikYml, Content: buildTraefikStaticConfig(cfg, tcpListens), ContentMode: contentModeSync},
		{SourceType: mountSourceLogical, Source: "acme.json", Target: gatewayMountTargetAcmeJSON, Content: "{}", ContentMode: contentModeSeed},
	}
}

func mergeManagedGatewayMounts(existing []mountSpec, managed []mountSpec) []mountSpec {
	managedTargets := map[string]struct{}{
		gatewayMountTargetDockerSock: {}, gatewayMountTargetTraefikYml: {}, gatewayMountTargetAcmeJSON: {},
	}
	out := make([]mountSpec, 0, len(existing)+len(managed))
	for _, mount := range existing {
		if _, isManaged := managedTargets[mount.Target]; !isManaged {
			out = append(out, mount)
		}
	}
	return append(out, managed...)
}

func upsertComponentByName(existing []model.VersionComponent, managed model.VersionComponent) []model.VersionComponent {
	out := make([]model.VersionComponent, 0, len(existing)+1)
	replaced := false
	for _, component := range existing {
		if component.Name == managed.Name {
			managed.Id = component.Id
			out = append(out, managed)
			replaced = true
			continue
		}
		out = append(out, component)
	}
	if !replaced {
		out = append(out, managed)
	}
	return out
}

func buildTraefikStaticConfig(cfg model.GatewayConfig, tcpListens []int) string {
	var builder strings.Builder
	builder.WriteString(strings.TrimSpace(`
api:
  dashboard: true
  insecure: true

entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"
`))
	builder.WriteByte('\n')
	for _, listen := range tcpListens {
		builder.WriteString("  ")
		builder.WriteString(tcpEntrypointName(listen))
		builder.WriteString(":\n    address: \":")
		builder.WriteString(strconv.Itoa(listen))
		builder.WriteString("\"\n")
	}
	builder.WriteString(strings.TrimSpace(`
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
	builder.WriteByte('\n')
	if strings.EqualFold(strings.TrimSpace(cfg.TLSMode), "letsencrypt") {
		builder.WriteString(strings.TrimSpace(`
certificatesResolvers:
  letsencrypt:
    acme:
      email: admin@localhost
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web
`))
		builder.WriteByte('\n')
	}
	return builder.String()
}

// CompiledTCPListens reads non-reserved host TCP ports from the managed Traefik component.
func CompiledTCPListens(components []model.VersionComponent) []int {
	for _, component := range components {
		if component.Name != gatewayManagedComponentName {
			continue
		}
		var ports []string
		if component.PortsJSON == nil || json.Unmarshal([]byte(*component.PortsJSON), &ports) != nil {
			return nil
		}
		listens := make([]int, 0, len(ports))
		for _, port := range ports {
			parts := strings.Split(port, ":")
			if len(parts) != 2 {
				continue
			}
			listen, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err == nil {
				listens = append(listens, listen)
			}
		}
		return normalizeTCPListens(listens)
	}
	return nil
}

// TCPListensEqual compares host TCP port sets after normalization.
func TCPListensEqual(left []int, right []int) bool {
	left = normalizeTCPListens(left)
	right = normalizeTCPListens(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
