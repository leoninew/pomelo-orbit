package model

import (
	"fmt"
	"regexp"
	"strings"
)

var dnsLabel = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

// IsDNSLabel reports whether value can be used as one lower-case DNS label.
func IsDNSLabel(value string) bool {
	return dnsLabel.MatchString(value)
}

// RuntimeContainerName returns the explicit Docker container name emitted for
// an Application component in the generated Compose configuration.
func RuntimeContainerName(applicationCode, componentName string) string {
	name := strings.ToLower(strings.TrimSpace(applicationCode) + "-" + strings.TrimSpace(componentName))
	name = strings.Map(func(char rune) rune {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			return char
		}
		return '-'
	}, name)
	return strings.Trim(name, "-")
}

// GatewayNetworkName returns the shared Docker bridge network used by the
// Gateway and every Service which opts into Traefik routing.
func GatewayNetworkName() string {
	return "traefik"
}

// GatewayComponentName is the fixed product component name for managed Gateway Versions.
func GatewayComponentName() string {
	return "traefik"
}

// GatewayInitialPullPolicy is the pull policy written into the initial Gateway Version.
func GatewayInitialPullPolicy() string {
	return "missing"
}

// DeriveServiceComponentHost returns the public host for one Service component.
func DeriveServiceComponentHost(gateway *GatewayConfig, service Service, componentName string) (string, error) {
	if gateway == nil {
		return "", fmt.Errorf("gateway config is required")
	}
	if !IsDNSLabel(service.Code) {
		return "", fmt.Errorf("service code %q is invalid", service.Code)
	}
	if !IsDNSLabel(componentName) {
		return "", fmt.Errorf("component name %q is not a valid DNS label", componentName)
	}
	if !isDNSName(gateway.BaseDomain) {
		return "", fmt.Errorf("gateway base_domain %q is invalid", gateway.BaseDomain)
	}
	host := componentName + "." + service.Code + "." + gateway.BaseDomain
	if len(host) > 253 {
		return "", fmt.Errorf("derived host %q is too long", host)
	}
	return host, nil
}

func isDNSName(value string) bool {
	if value == "" || len(value) > 253 {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if !IsDNSLabel(label) {
			return false
		}
	}
	return true
}
