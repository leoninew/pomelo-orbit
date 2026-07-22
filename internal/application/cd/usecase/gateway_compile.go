package cdsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const (
	// gatewayManagedComponentName is the Version component upserted by compile.
	gatewayManagedComponentName = "traefik"
	// defaultGatewayImage is used when gateway_config.image is empty.
	defaultGatewayImage = "traefik:v3.6"
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
func (s Service) CompileGatewayToVersion(ctx context.Context, app model.Application, cfg model.GatewayConfig) (string, error) {
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
	managed, err := buildManagedGatewayComponent(cfg, existing)
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

func buildManagedGatewayComponent(cfg model.GatewayConfig, existing []model.VersionComponent) (model.VersionComponent, error) {
	image := defaultGatewayImage
	if cfg.Image != nil && strings.TrimSpace(*cfg.Image) != "" {
		image = strings.TrimSpace(*cfg.Image)
	}
	portsJSON := `["80:80","443:443","8080:8080"]`
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
	mounts := mergeManagedGatewayMounts(priorMounts, buildManagedGatewayMounts())
	mountsRaw, err := json.Marshal(mounts)
	if err != nil {
		return model.VersionComponent{}, fmt.Errorf("marshal mounts: %w", err)
	}
	mountsStr := string(mountsRaw)
	portsStr := portsJSON
	return model.VersionComponent{
		Name:       gatewayManagedComponentName,
		Image:      image,
		PortsJSON:  &portsStr,
		MountsJSON: &mountsStr,
	}, nil
}

func buildManagedGatewayMounts() []MountSpec {
	yml := buildTraefikStaticConfig()
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
			// Keep id if present for stability; ReplaceVersionComponents reassigns via caller.
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

// buildTraefikStaticConfig generates a minimal Traefik static file with rest + docker providers.
func buildTraefikStaticConfig() string {
	// entryPoints names align with Environment.default_entrypoint defaults (web / websecure).
	// providers.rest + api.insecure enable platform PUT/GET on :8080.
	// docker provider watches sock; network name matches defaultGatewayNetworkName.
	return strings.TrimSpace(`
api:
  dashboard: true
  insecure: true

entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"

providers:
  docker:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false
    network: traefik
  rest:
    insecure: true

log:
  level: INFO
`) + "\n"
}
