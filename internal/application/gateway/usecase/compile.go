package gatewaysvc

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

const (
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
		tcpListens = CompiledTCPListens(existing, managedGatewayComponentName)
	}
	managed, err := buildManagedGatewayComponent(cfg, existing, normalizeTCPListens(tcpListens), managedGatewayComponentName, managedGatewayNetworkName)
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
	// ReplaceVersionComponents deletes version_component rows first; FK CASCADE
	// removes service_component mappings for Services already bound to this Version.
	if err := s.serviceCommands.AlignComponentMappingsToVersion(ctx, app.Id, version.Id); err != nil {
		return "", err
	}
	return version.Id, nil
}

// CompileTCPRouteListeners compiles the static Gateway entrypoints required by
// the complete enabled custom TCP Route listener set.
func (s Service) CompileTCPRouteListeners(ctx context.Context, cfg model.GatewayConfig, listens []int) error {
	app, err := s.application.Application(ctx, cfg.ApplicationId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load gateway application for TCP routes", err)
	}
	versions, err := s.application.ListVersions(ctx, app.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to list gateway versions for TCP routes", err)
	}
	for _, version := range versions {
		if version.Status != status.VersionStatusUnpublished {
			continue
		}
		components, err := s.application.VersionComponentsByVersion(ctx, version.Id)
		if err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to load gateway components for TCP routes", err)
		}
		if TCPListensEqual(CompiledTCPListens(components, managedGatewayComponentName), listens) {
			return nil
		}
		break
	}
	if _, err := s.CompileGatewayToVersion(ctx, app, cfg, listens...); err != nil {
		return err
	}
	return nil
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
	var source *model.Version
	for index := range versions {
		if versions[index].Status == status.VersionStatusPublished {
			source = &versions[index]
			break
		}
	}
	if source == nil {
		return model.Version{}, apperror.New(apperror.KindValidation, "gateway has no source version to compile")
	}
	version := model.Version{
		Id:                   idutil.NewId(),
		ApplicationId:        applicationId,
		Label:                s.traefik.Image + "-" + idutil.NewId(),
		Status:               status.VersionStatusUnpublished,
		CreatedFromVersionId: &source.Id,
	}
	components, err := s.application.VersionComponentsByVersion(ctx, source.Id)
	if err != nil {
		return model.Version{}, apperror.Wrap(apperror.KindInternal, "Failed to load gateway source components", err)
	}
	for index := range components {
		components[index].Id = idutil.NewId()
		components[index].VersionId = version.Id
	}
	if err := s.application.CreateVersion(ctx, version); err != nil {
		return model.Version{}, apperror.Wrap(apperror.KindInternal, "Failed to create managed version", err)
	}
	if err := s.application.ReplaceVersionComponents(ctx, version.Id, components); err != nil {
		return model.Version{}, apperror.Wrap(apperror.KindInternal, "Failed to copy gateway source components", err)
	}
	return version, nil
}

func buildManagedGatewayComponent(cfg model.GatewayConfig, existing []model.VersionComponent, tcpListens []int, componentName string, networkName string) (model.VersionComponent, error) {
	var previous *model.VersionComponent
	for index := range existing {
		if existing[index].Name == componentName {
			previous = &existing[index]
			break
		}
	}
	if previous == nil || strings.TrimSpace(previous.Image) == "" {
		return model.VersionComponent{}, fmt.Errorf("gateway version must declare a %s component image", componentName)
	}
	endpoints := []model.VersionComponentEndpoint{
		{Protocol: "tcp", ContainerPort: 80, Mode: "host", BindAddress: stringRef("0.0.0.0"), ListenPort: intRef(80)},
		{Protocol: "tcp", ContainerPort: 443, Mode: "host", BindAddress: stringRef("0.0.0.0"), ListenPort: intRef(443)},
		{Protocol: "http", ContainerPort: 8080, Mode: "local", BindAddress: stringRef("127.0.0.1"), ListenPort: intRef(8080)},
	}
	for _, listen := range tcpListens {
		endpoints = append(endpoints, model.VersionComponentEndpoint{Protocol: "tcp", ContainerPort: listen, Mode: "host", BindAddress: stringRef("0.0.0.0"), ListenPort: intRef(listen)})
	}
	mounts := mergeManagedGatewayMounts(previous.Mounts, buildManagedGatewayMounts(cfg, tcpListens, networkName))
	return model.VersionComponent{
		Id: previous.Id, Name: componentName, Image: previous.Image,
		PullPolicy: previous.PullPolicy, Endpoints: endpoints, Mounts: mounts,
	}, nil
}

func buildManagedGatewayMounts(cfg model.GatewayConfig, tcpListens []int, networkName string) []model.VersionComponentMount {
	return []model.VersionComponentMount{
		{SourceType: mountSourceFile, Source: gatewayDockerSocketPath, SourceIsHostPath: true, Target: gatewayDockerSocketPath, ReadOnly: true},
		{SourceType: mountSourceControlledFile, Source: "traefik.yml", Target: gatewayMountTargetTraefikYml, Content: buildTraefikStaticConfig(cfg, tcpListens, networkName), Mode: "0644"},
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

func buildTraefikStaticConfig(cfg model.GatewayConfig, tcpListens []int, networkName string) string {
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
	builder.WriteString("providers:\n  docker:\n    endpoint: \"unix:///var/run/docker.sock\"\n    exposedByDefault: false\n    network: ")
	builder.WriteString(networkName)
	builder.WriteString("\n  rest:\n    insecure: true\n\nlog:\n  level: INFO\n")
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
func CompiledTCPListens(components []model.VersionComponent, componentName string) []int {
	for _, component := range components {
		if component.Name != componentName {
			continue
		}
		listens := make([]int, 0, len(component.Endpoints))
		for _, endpoint := range component.Endpoints {
			if endpoint.ListenPort != nil {
				listens = append(listens, *endpoint.ListenPort)
			}
		}
		return normalizeTCPListens(listens)
	}
	return nil
}

func stringRef(value string) *string { return &value }
func intRef(value int) *int          { return &value }

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
