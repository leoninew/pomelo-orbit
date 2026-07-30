package gatewaysvc

import (
	"context"
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
	gatewayDockerSocketPath      = "/var/run/docker.sock"
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
	if tcpListens == nil {
		tcpListens = CompiledTCPListens(existing)
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
	ports := []model.VersionComponentPort{
		{HostPort: 80, ContainerPort: 80},
		{HostPort: 443, ContainerPort: 443},
		{HostPort: 8080, ContainerPort: 8080},
	}
	for _, listen := range tcpListens {
		ports = append(ports, model.VersionComponentPort{HostPort: listen, ContainerPort: listen})
	}
	var previousMounts []model.VersionComponentMount
	for _, component := range existing {
		if component.Name != gatewayManagedComponentName {
			continue
		}
		previousMounts = component.Mounts
		break
	}
	mounts := mergeManagedGatewayMounts(previousMounts, buildManagedGatewayMounts(cfg, tcpListens))
	return model.VersionComponent{
		Name: gatewayManagedComponentName, Image: strings.TrimSpace(*cfg.Image),
		Ports: ports, Mounts: mounts,
	}, nil
}

func buildManagedGatewayMounts(cfg model.GatewayConfig, tcpListens []int) []model.VersionComponentMount {
	return []model.VersionComponentMount{
		{SourceType: mountSourceFile, Source: gatewayDockerSocketPath, SourceIsHostPath: true, Target: gatewayDockerSocketPath, ReadOnly: true},
		{SourceType: mountSourceControlledFile, Source: "traefik.yml", Target: gatewayMountTargetTraefikYml, Content: buildTraefikStaticConfig(cfg, tcpListens), Mode: "0644"},
		{SourceType: mountSourceControlledFile, Source: "acme.json", Target: gatewayMountTargetAcmeJSON, Content: "{}", Mode: "0600", IgnoreIfExists: true},
	}
}

func mergeManagedGatewayMounts(existing []model.VersionComponentMount, managed []model.VersionComponentMount) []model.VersionComponentMount {
	managedTargets := map[string]struct{}{
		gatewayDockerSocketPath: {}, gatewayMountTargetTraefikYml: {}, gatewayMountTargetAcmeJSON: {},
	}
	out := make([]model.VersionComponentMount, 0, len(existing)+len(managed))
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
		listens := make([]int, 0, len(component.Ports))
		for _, port := range component.Ports {
			listens = append(listens, port.HostPort)
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
