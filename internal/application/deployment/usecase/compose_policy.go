package deploymentsvc

import (
	"fmt"
	"strings"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

type composePolicy struct {
	gatewayCarrier      bool
	joinNetwork         bool
	networkName         string
	dashboardHost       string
	dashboardEntrypoint string
	ports               []publishedPort
	routes              []publishedRoute
}

type publishedPort struct {
	component     string
	address       string
	hostPort      int
	containerPort int
}

type publishedRoute struct {
	component     string
	containerPort int
	host          string
	pathPrefix    string
	entrypoint    string
}

// resolveComposePolicy decides exposure and network membership before encoding.
func resolveComposePolicy(plan model.EffectiveServicePlan) (composePolicy, error) {
	policy := composePolicy{gatewayCarrier: isGatewayCarrier(plan)}
	switch plan.Application.Kind {
	case status.ApplicationKindStandard, status.ApplicationKindGateway:
	default:
		return composePolicy{}, fmt.Errorf("unsupported application kind %q", plan.Application.Kind)
	}
	policy.joinNetwork = policy.gatewayCarrier || plan.JoinsTraefikNetwork()
	if policy.joinNetwork {
		if plan.Gateway == nil || strings.TrimSpace(plan.Gateway.NetworkName) == "" {
			return composePolicy{}, fmt.Errorf("project gateway network is required when joining the Traefik network")
		}
		policy.networkName = plan.Gateway.NetworkName
	}
	if policy.gatewayCarrier {
		host, err := deriveHost(plan.Gateway, plan.Application.Code)
		if err != nil {
			return composePolicy{}, err
		}
		if plan.Gateway.DefaultEntrypoint == "" {
			return composePolicy{}, fmt.Errorf("gateway default_entrypoint is required")
		}
		policy.dashboardHost, policy.dashboardEntrypoint = host, plan.Gateway.DefaultEntrypoint
	}
	httpRoutes := map[string]struct{}{}
	for _, component := range plan.Components {
		for _, endpoint := range component.Endpoints {
			displayName := model.EndpointDisplayName(endpoint.Protocol, endpoint.ContainerPort)
			switch endpoint.Mode {
			case "internal":
				continue
			case "local", "host":
				if endpoint.ListenPort == nil {
					return composePolicy{}, fmt.Errorf("endpoint %s/%s requires listen_port", component.Name, displayName)
				}
				address := "0.0.0.0"
				if endpoint.Mode == "local" {
					address = "127.0.0.1"
				}
				if endpoint.BindAddress != nil && *endpoint.BindAddress != "" {
					address = *endpoint.BindAddress
				}
				if endpoint.Mode == "local" && address != "127.0.0.1" && address != "::1" {
					return composePolicy{}, fmt.Errorf("local endpoint %s/%s must bind loopback", component.Name, displayName)
				}
				policy.ports = append(policy.ports, publishedPort{component.Name, address, *endpoint.ListenPort, endpoint.ContainerPort})
			case "gateway":
				if !policy.joinNetwork {
					continue
				}
				if endpoint.Protocol != "http" {
					return composePolicy{}, fmt.Errorf("gateway endpoint %s/%s must use http", component.Name, displayName)
				}
				host, err := model.DeriveServiceComponentHost(plan.Gateway, plan.Service, component.Name)
				if err != nil {
					return composePolicy{}, err
				}
				entrypoint := plan.Gateway.DefaultEntrypoint
				if endpoint.Entrypoint != nil && *endpoint.Entrypoint != "" {
					entrypoint = *endpoint.Entrypoint
				}
				if entrypoint == "" {
					return composePolicy{}, fmt.Errorf("gateway endpoint %s/%s requires entrypoint", component.Name, displayName)
				}
				pathPrefix := normalizedPathPrefix(endpoint.PathPrefix)
				routeKey := host + "\x00" + entrypoint + "\x00" + pathPrefix
				if _, exists := httpRoutes[routeKey]; exists {
					return composePolicy{}, fmt.Errorf("gateway endpoint %s/%s duplicates route %s%s", component.Name, displayName, host, pathPrefix)
				}
				httpRoutes[routeKey] = struct{}{}
				policy.routes = append(policy.routes, publishedRoute{component.Name, endpoint.ContainerPort, host, pathPrefix, entrypoint})
			default:
				return composePolicy{}, fmt.Errorf("endpoint %s/%s has unsupported mode %q", component.Name, displayName, endpoint.Mode)
			}
		}
	}
	return policy, nil
}

func normalizedPathPrefix(value *string) string {
	if value == nil || *value == "" || *value == "/" {
		return "/"
	}
	return *value
}

func deriveHost(gateway *model.GatewayConfig, appCode string) (string, error) {
	if gateway == nil || gateway.InternalDomain == "" {
		return "", fmt.Errorf("gateway internal_domain is required")
	}
	if appCode == "" {
		return "", fmt.Errorf("application code is required")
	}
	host := appCode + "." + gateway.InternalDomain
	if host == "" || len(host) > 253 || strings.ContainsAny(host, " `\t\r\n") {
		return "", fmt.Errorf("invalid derived host %q", host)
	}
	return host, nil
}
