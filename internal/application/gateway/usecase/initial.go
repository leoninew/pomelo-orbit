package gatewaysvc

import (
	"strconv"
	"strings"

	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/config"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

const (
	gatewayDockerSocketPath      = "/var/run/docker.sock"
	gatewayMountTargetTraefikYml = "/etc/traefik/traefik.yml"
	gatewayMountTargetCertDir    = "/etc/traefik/certs"
	gatewayMountTargetAcmeDir    = "/letsencrypt"
	mountSourceDirectory         = "directory"
	mountSourceFile              = "file"
	mountSourceControlledFile    = "controlled_file"
)

// buildInitialGatewayComponent creates the first editable Version declaration.
// Later Gateway, Route, and deployment operations deliberately do not call it.
func buildInitialGatewayComponent(versionID, image, pullPolicy string, gateway model.GatewayConfig, cert config.CertConfig, certDirectory string) model.VersionComponent {
	return model.VersionComponent{
		Id:         idutil.NewId(),
		VersionId:  versionID,
		Name:       managedGatewayComponentName,
		Image:      image,
		PullPolicy: pullPolicy,
		Endpoints: []model.VersionComponentEndpoint{
			{Protocol: "tcp", ContainerPort: 80, Mode: "host", BindAddress: stringRef("0.0.0.0"), ListenPort: intRef(80)},
			{Protocol: "tcp", ContainerPort: 443, Mode: "host", BindAddress: stringRef("0.0.0.0"), ListenPort: intRef(443)},
			{Protocol: "http", ContainerPort: 8080, Mode: "local", BindAddress: stringRef("127.0.0.1"), ListenPort: intRef(8080)},
		},
		Mounts: []model.VersionComponentMount{
			{SourceType: mountSourceFile, Source: gatewayDockerSocketPath, SourceIsHostPath: true, Target: gatewayDockerSocketPath, ReadOnly: true},
			{SourceType: mountSourceControlledFile, Source: "traefik.yml", Target: gatewayMountTargetTraefikYml, Content: buildInitialTraefikStaticConfig(gateway, cert), Mode: "0644"},
			{SourceType: mountSourceDirectory, Source: certDirectory, SourceIsHostPath: true, Target: gatewayMountTargetCertDir},
			{SourceType: mountSourceDirectory, Source: certDirectory, SourceIsHostPath: true, Target: gatewayMountTargetAcmeDir},
		},
	}
}

func buildInitialTraefikStaticConfig(gateway model.GatewayConfig, cert config.CertConfig) string {
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
	if strings.EqualFold(strings.TrimSpace(gateway.TLSMode), "letsencrypt") {
		builder.WriteString("certificatesResolvers:\n  letsencrypt:\n    acme:\n      email: ")
		builder.WriteString(strconv.Quote(strings.TrimSpace(cert.LetsEncrypt.Email)))
		builder.WriteString("\n      storage: /letsencrypt/acme.json\n")
		if strings.EqualFold(strings.TrimSpace(cert.LetsEncrypt.Challenge), "dns") {
			builder.WriteString("      dnsChallenge:\n        provider: ")
			builder.WriteString(strconv.Quote(strings.TrimSpace(cert.LetsEncrypt.DNSProvider)))
			builder.WriteByte('\n')
		} else {
			builder.WriteString("      httpChallenge:\n        entryPoint: web\n")
		}
	}
	return builder.String()
}

func stringRef(value string) *string { return &value }
func intRef(value int) *int          { return &value }
