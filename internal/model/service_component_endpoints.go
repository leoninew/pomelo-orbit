package model

import "fmt"

// MergeServiceComponentEndpoints resolves endpoint overlays without evaluating
// environment values or other deployment configuration.
func MergeServiceComponentEndpoints(declaration VersionComponent, overlay ServiceComponent) ([]VersionComponentEndpoint, error) {
	byIdentity := make(map[string]ServiceComponentEndpoint, len(overlay.Endpoints))
	for _, item := range overlay.Endpoints {
		identity := EndpointDisplayName(item.Protocol, item.ContainerPort)
		if _, exists := byIdentity[identity]; exists {
			return nil, fmt.Errorf("component %s has duplicate endpoint overlay %s", declaration.Name, identity)
		}
		byIdentity[identity] = item
	}
	endpoints := make([]VersionComponentEndpoint, 0, len(declaration.Endpoints))
	for _, item := range declaration.Endpoints {
		identity := EndpointDisplayName(item.Protocol, item.ContainerPort)
		merged, exists := byIdentity[identity]
		if !exists {
			endpoints = append(endpoints, item)
			continue
		}
		switch merged.State {
		case ServiceComponentOverlayOverride:
			if merged.Mode != nil {
				item.Mode = *merged.Mode
			}
			if merged.BindAddress != nil {
				value := *merged.BindAddress
				item.BindAddress = &value
			}
			if merged.ListenPort != nil {
				value := *merged.ListenPort
				item.ListenPort = &value
			}
			if merged.Entrypoint != nil {
				value := *merged.Entrypoint
				item.Entrypoint = &value
			}
			if merged.PathPrefix != nil {
				value := *merged.PathPrefix
				item.PathPrefix = &value
			}
		case ServiceComponentOverlayDeleted:
			item.Mode, item.BindAddress, item.ListenPort, item.Entrypoint, item.PathPrefix = "internal", nil, nil, nil, nil
		default:
			return nil, fmt.Errorf("component %s endpoint overlay %s has invalid state", declaration.Name, identity)
		}
		endpoints = append(endpoints, item)
		delete(byIdentity, identity)
	}
	if len(byIdentity) != 0 {
		return nil, fmt.Errorf("component %s overlays an undeclared endpoint", declaration.Name)
	}
	return endpoints, nil
}
