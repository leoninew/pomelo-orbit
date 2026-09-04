package gatewaysvc

import (
	"strings"

	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

const (
	gatewayVersionRoleBase    = "base"
	gatewayVersionProfileHTTP = "http"
	gatewayVersionProfileDNS  = "dns"
	gatewayVersionProfileBoth = "http-dns"

	gatewayDockerSocketPath      = "/var/run/docker.sock"
	gatewayMountTargetTraefikYml = "/etc/traefik/traefik.yml"
	gatewayMountTargetCertDir    = "/etc/traefik/certs"
	gatewayMountTargetAcmeDir    = "/letsencrypt"
)

type initialGatewayVersion struct {
	Role      string
	Version   model.Version
	Component model.VersionComponent
}

func buildInitialGatewayVersions(applicationID, image, pullPolicy, componentName, networkName string) []initialGatewayVersion {
	roles := []string{gatewayVersionRoleBase, gatewayVersionProfileHTTP, gatewayVersionProfileDNS, gatewayVersionProfileBoth}
	items := make([]initialGatewayVersion, 0, len(roles))
	for _, role := range roles {
		label := image
		if role != gatewayVersionRoleBase {
			label += " (" + role + ")"
		}
		version := model.Version{
			Id:               idutil.NewId(),
			ApplicationId:    applicationID,
			Label:            label,
			Status:           "unpublished",
			ComponentSummary: componentName,
		}
		items = append(items, initialGatewayVersion{
			Role:      role,
			Version:   version,
			Component: buildInitialGatewayComponent(version.Id, image, pullPolicy, componentName, role, networkName),
		})
	}
	return items
}

// buildInitialGatewayComponent creates an ordinary Version declaration. The
// resolver layout is fixed per profile and never derived from GatewayConfig.
func buildInitialGatewayComponent(versionID, image, pullPolicy, componentName, role, networkName string) model.VersionComponent {
	return model.VersionComponent{
		Id:         idutil.NewId(),
		VersionId:  versionID,
		Name:       componentName,
		Image:      image,
		PullPolicy: pullPolicy,
		Endpoints: []model.VersionComponentEndpoint{
			{Protocol: "tcp", ContainerPort: 80, Mode: "host", BindAddress: stringRef("0.0.0.0"), ListenPort: intRef(80)},
			{Protocol: "tcp", ContainerPort: 443, Mode: "host", BindAddress: stringRef("0.0.0.0"), ListenPort: intRef(443)},
			{Protocol: "http", ContainerPort: 8080, Mode: "local", BindAddress: stringRef("127.0.0.1"), ListenPort: intRef(8080)},
		},
		Mounts: []model.VersionComponentMount{
			{SourceType: "file", Source: gatewayDockerSocketPath, SourceIsHostPath: true, Target: gatewayDockerSocketPath, ReadOnly: true},
			{SourceType: "controlled_file", Source: "./traefik.yml", Target: gatewayMountTargetTraefikYml, Content: initialTraefikStaticConfig(role, networkName), Mode: "0644"},
			{SourceType: "directory", Source: "./gateway/certs", Target: gatewayMountTargetCertDir},
			{SourceType: "directory", Source: "./gateway/acme", Target: gatewayMountTargetAcmeDir},
		},
	}
}

func initialTraefikStaticConfig(role, networkName string) string {
	var builder strings.Builder
	builder.WriteString(`api:
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
    network: ` + networkName + `
  rest:
    insecure: true

log:
  level: INFO
`)
	builder.WriteByte('\n')
	switch role {
	case gatewayVersionProfileHTTP:
		builder.WriteString(acmeHTTPResolver)
	case gatewayVersionProfileDNS:
		builder.WriteString(acmeDNSResolver)
	case gatewayVersionProfileBoth:
		builder.WriteString(acmeHTTPResolver)
		builder.WriteString("\n")
		builder.WriteString(strings.TrimPrefix(acmeDNSResolver, "certificatesResolvers:\n"))
	}
	return builder.String()
}

const acmeHTTPResolver = `certificatesResolvers:
  letsencrypt:
    acme:
      email: ""
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web
`

const acmeDNSResolver = `certificatesResolvers:
  letsencrypt-dns:
    acme:
      email: ""
      storage: /letsencrypt/acme.json
      dnsChallenge:
        provider: cloudflare
        resolvers:
          - "1.1.1.1:53"
          - "8.8.8.8:53"
        propagation:
          delayBeforeChecks: 60s
`

func stringRef(value string) *string { return &value }
func intRef(value int) *int          { return &value }

func buildInitialGatewayDashboardRoute(app model.Application, cfg model.GatewayConfig) model.Route {
	projectID := *app.ProjectId
	return model.Route{
		Id:            idutil.NewId(),
		ProjectId:     &projectID,
		Name:          "traefik",
		Protocol:      "http",
		Domain:        "traefik-dashboard." + cfg.BaseDomain,
		PathPrefix:    "/",
		TargetUrl:     "http://" + model.RuntimeContainerName(app.Code, cfg.TraefikComponentName) + ":8080",
		Enabled:       false,
		HTTPSEnabled:  false,
		CertType:      "manual",
		AcmeChallenge: "http",
	}
}
