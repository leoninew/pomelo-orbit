package deploymentsvc

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

const versionPreviewInstanceKey = "default"

// BuildVersionPreviewPlan creates a compose preview from version declarations
// only. It does not load or merge Service runtime overlays.
func BuildVersionPreviewPlan(app model.Application, version model.Version, declarations []model.VersionComponent, gateway *model.GatewayConfig) (model.EffectiveServicePlan, error) {
	if version.ApplicationId != app.Id {
		return model.EffectiveServicePlan{}, fmt.Errorf("version %s does not belong to application %s", version.Id, app.Id)
	}
	plan := model.EffectiveServicePlan{
		Application: app,
		Version:     version,
		Service: model.Service{
			ApplicationId: app.Id,
			VersionId:     version.Id,
			InstanceKey:   versionPreviewInstanceKey,
			Code:          app.Code + "-preview",
		},
		Gateway:    gateway,
		Components: make([]model.EffectiveServiceComponent, 0, len(declarations)),
	}
	for _, declaration := range declarations {
		plan.Components = append(plan.Components, effectiveComponentFromVersion(declaration))
	}
	return plan, nil
}

// BuildEffectiveServicePlan merges one Service's sparse overlays with its
// Version declarations. Both preview and execution call this function.
func BuildEffectiveServicePlan(app model.Application, version model.Version, service model.Service, declarations []model.VersionComponent, overlays []model.ServiceComponent, env []model.ServiceEnv, gateway *model.GatewayConfig) (model.EffectiveServicePlan, string, error) {
	if service.VersionId != version.Id {
		return model.EffectiveServicePlan{}, "", fmt.Errorf("service %s does not select version %s", service.Id, version.Id)
	}
	serviceEnv, err := serviceEnvironmentValues(env)
	if err != nil {
		return model.EffectiveServicePlan{}, "", err
	}
	overlayBySource := make(map[string]model.ServiceComponent, len(overlays))
	for _, overlay := range overlays {
		if overlay.ServiceId != service.Id {
			return model.EffectiveServicePlan{}, "", fmt.Errorf("service component %s does not belong to service", overlay.Id)
		}
		if _, duplicate := overlayBySource[overlay.SourceVersionComponentId]; duplicate {
			return model.EffectiveServicePlan{}, "", fmt.Errorf("duplicate service component mapping for %s", overlay.SourceVersionComponentId)
		}
		overlayBySource[overlay.SourceVersionComponentId] = overlay
	}
	plan := model.EffectiveServicePlan{Application: app, Version: version, Service: service, Gateway: gateway, Components: make([]model.EffectiveServiceComponent, 0, len(declarations))}
	for _, declaration := range declarations {
		overlay, found := overlayBySource[declaration.Id]
		if !found {
			return model.EffectiveServicePlan{}, "", fmt.Errorf("service %s has no component mapping for %s", service.Id, declaration.Name)
		}
		if overlay.ComponentName != declaration.Name {
			return model.EffectiveServicePlan{}, "", fmt.Errorf("service component %s no longer matches declaration %s", overlay.Id, declaration.Id)
		}
		component, err := MergeServiceComponent(declaration, overlay, serviceEnv)
		if err != nil {
			return model.EffectiveServicePlan{}, "", err
		}
		plan.Components = append(plan.Components, component)
	}
	if len(overlays) != len(declarations) {
		return model.EffectiveServicePlan{}, "", fmt.Errorf("service %s component mapping does not match selected version", service.Id)
	}
	hash, err := EffectiveServicePlanHash(plan)
	if err != nil {
		return model.EffectiveServicePlan{}, "", err
	}
	return plan, hash, nil
}

// MergeServiceComponent applies one sparse ServiceComponent overlay to a
// VersionComponent declaration. It is shared by the service detail read model
// and whole-service plan construction.
func MergeServiceComponent(declaration model.VersionComponent, overlay model.ServiceComponent, serviceEnv map[string]string) (model.EffectiveServiceComponent, error) {
	result := effectiveComponentFromVersion(declaration)
	result.ServiceComponentId = overlay.Id
	if overlay.Entrypoint != nil {
		result.Entrypoint = cloneStringSlice(overlay.Entrypoint)
	}
	if overlay.Command != nil {
		result.Command = cloneStringSlice(overlay.Command)
	}
	if overlay.PullPolicy != nil {
		result.PullPolicy = *overlay.PullPolicy
	}
	if overlay.RestartPolicy != nil {
		result.RestartPolicy = cloneString(overlay.RestartPolicy)
	}
	// Rebuild fields that accept sparse runtime overlays. Keeping the declaration
	// values here would emit both the original and the override.
	result.Env = make([]model.VersionComponentEnv, 0, len(declaration.Env))
	result.Mounts = make([]model.VersionComponentMount, 0, len(declaration.Mounts))
	result.Endpoints = make([]model.VersionComponentEndpoint, 0, len(declaration.Endpoints))
	envOverlay := make(map[string]model.ServiceComponentEnv, len(overlay.Env))
	for _, item := range overlay.Env {
		if _, exists := envOverlay[item.Key]; exists {
			return result, fmt.Errorf("component %s has duplicate environment overlay %s", declaration.Name, item.Key)
		}
		envOverlay[item.Key] = item
	}
	for _, item := range declaration.Env {
		merged, exists := envOverlay[item.Key]
		if !exists {
			value, err := resolveServiceEnvironment(item.Value, serviceEnv)
			if err != nil {
				return result, fmt.Errorf("component %s environment %s: %w", declaration.Name, item.Key, err)
			}
			item.Value = value
			result.Env = append(result.Env, item)
			continue
		}
		switch merged.State {
		case model.ServiceComponentOverlayOverride:
			if merged.Value == nil {
				return result, fmt.Errorf("component %s environment overlay %s has no value", declaration.Name, item.Key)
			}
			result.Env = append(result.Env, model.VersionComponentEnv{Key: item.Key, Value: *merged.Value})
		case model.ServiceComponentOverlayDeleted:
		default:
			return result, fmt.Errorf("component %s environment overlay %s has invalid state", declaration.Name, item.Key)
		}
		delete(envOverlay, item.Key)
	}
	if len(envOverlay) != 0 {
		return result, fmt.Errorf("component %s overlays an undeclared environment value", declaration.Name)
	}

	mountOverlay := make(map[string]model.ServiceComponentMount, len(overlay.Mounts))
	for _, item := range overlay.Mounts {
		if _, exists := mountOverlay[item.Target]; exists {
			return result, fmt.Errorf("component %s has duplicate mount overlay %s", declaration.Name, item.Target)
		}
		mountOverlay[item.Target] = item
	}
	for _, item := range declaration.Mounts {
		merged, exists := mountOverlay[item.Target]
		if !exists {
			result.Mounts = append(result.Mounts, item)
			continue
		}
		switch merged.State {
		case model.ServiceComponentOverlayOverride:
			if merged.Source == nil || merged.SourceIsHostPath == nil {
				return result, fmt.Errorf("component %s mount overlay %s is incomplete", declaration.Name, item.Target)
			}
			item.Source = *merged.Source
			item.SourceIsHostPath = *merged.SourceIsHostPath
			result.Mounts = append(result.Mounts, item)
		case model.ServiceComponentOverlayDeleted:
		default:
			return result, fmt.Errorf("component %s mount overlay %s has invalid state", declaration.Name, item.Target)
		}
		delete(mountOverlay, item.Target)
	}
	if len(mountOverlay) != 0 {
		return result, fmt.Errorf("component %s overlays an undeclared mount", declaration.Name)
	}

	result.Resources = mergeServiceResources(declaration.Resources, overlay.Resources)
	if declaration.Resources == nil && overlay.Resources != nil {
		return result, fmt.Errorf("component %s overlays undeclared resources", declaration.Name)
	}

	endpointOverlay := make(map[string]model.ServiceComponentEndpoint, len(overlay.Endpoints))
	for _, item := range overlay.Endpoints {
		identity := model.EndpointDisplayName(item.Protocol, item.ContainerPort)
		if _, exists := endpointOverlay[identity]; exists {
			return result, fmt.Errorf("component %s has duplicate endpoint overlay %s", declaration.Name, identity)
		}
		endpointOverlay[identity] = item
	}
	for _, item := range declaration.Endpoints {
		identity := model.EndpointDisplayName(item.Protocol, item.ContainerPort)
		merged, exists := endpointOverlay[identity]
		if !exists {
			result.Endpoints = append(result.Endpoints, item)
			continue
		}
		switch merged.State {
		case model.ServiceComponentOverlayOverride:
			if merged.Mode != nil {
				item.Mode = *merged.Mode
			}
			if merged.BindAddress != nil {
				item.BindAddress = cloneString(merged.BindAddress)
			}
			if merged.ListenPort != nil {
				item.ListenPort = cloneInt(merged.ListenPort)
			}
			if merged.Entrypoint != nil {
				item.Entrypoint = cloneString(merged.Entrypoint)
			}
			if merged.PathPrefix != nil {
				item.PathPrefix = cloneString(merged.PathPrefix)
			}
		case model.ServiceComponentOverlayDeleted:
			item.Mode, item.BindAddress, item.ListenPort, item.Entrypoint, item.PathPrefix = "internal", nil, nil, nil, nil
		default:
			return result, fmt.Errorf("component %s endpoint overlay %s has invalid state", declaration.Name, identity)
		}
		result.Endpoints = append(result.Endpoints, item)
		delete(endpointOverlay, identity)
	}
	if len(endpointOverlay) != 0 {
		return result, fmt.Errorf("component %s overlays an undeclared endpoint", declaration.Name)
	}
	return result, nil
}

func serviceEnvironmentValues(env []model.ServiceEnv) (map[string]string, error) {
	if len(env) == 0 {
		return nil, nil
	}
	values := make(map[string]string, len(env))
	for _, item := range env {
		if _, exists := values[item.Key]; exists {
			return nil, fmt.Errorf("service has duplicate environment %s", item.Key)
		}
		values[item.Key] = item.Value
	}
	return values, nil
}

func resolveServiceEnvironment(value string, serviceEnv map[string]string) (string, error) {
	key, requirement, isPlaceholder := parsePlaceholder(value)
	if !isPlaceholder {
		return value, nil
	}
	if resolved, exists := serviceEnv[key]; exists && (!requirement.Required || resolved != "") {
		return resolved, nil
	}
	if requirement.HasDefault {
		return requirement.Default, nil
	}
	return "", fmt.Errorf("requires service environment %s", key)
}

func effectiveComponentFromVersion(declaration model.VersionComponent) model.EffectiveServiceComponent {
	return model.EffectiveServiceComponent{
		SourceComponentId: declaration.Id,
		Name:              declaration.Name,
		Image:             declaration.Image,
		Entrypoint:        cloneStringSlice(declaration.Entrypoint),
		Command:           cloneStringSlice(declaration.Command),
		Env:               append([]model.VersionComponentEnv(nil), declaration.Env...),
		Mounts:            append([]model.VersionComponentMount(nil), declaration.Mounts...),
		Dependencies:      append([]model.VersionComponentDependency(nil), declaration.Dependencies...),
		Healthcheck:       declaration.Healthcheck,
		Resources:         declaration.Resources,
		PullPolicy:        declaration.PullPolicy,
		RestartPolicy:     declaration.RestartPolicy,
		Tmpfs:             append([]model.VersionComponentTmpfs(nil), declaration.Tmpfs...),
		Ulimits:           append([]model.VersionComponentUlimit(nil), declaration.Ulimits...),
		Devices:           append([]model.VersionComponentDeviceRequest(nil), declaration.Devices...),
		Endpoints:         append([]model.VersionComponentEndpoint(nil), declaration.Endpoints...),
	}
}

func mergeServiceResources(base *model.VersionComponentResources, overlay *model.ServiceComponentResources) *model.VersionComponentResources {
	if base == nil || overlay == nil {
		return base
	}
	if overlay.State == model.ServiceComponentOverlayDeleted {
		return nil
	}
	result := *base
	if overlay.LimitCPUs != nil {
		result.LimitCPUs = cloneString(overlay.LimitCPUs)
	}
	if overlay.LimitMemory != nil {
		result.LimitMemory = cloneString(overlay.LimitMemory)
	}
	if overlay.ReservationCPUs != nil {
		result.ReservationCPUs = cloneString(overlay.ReservationCPUs)
	}
	if overlay.ReservationMemory != nil {
		result.ReservationMemory = cloneString(overlay.ReservationMemory)
	}
	return &result
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneStringSlice(value []string) []string {
	if value == nil {
		return nil
	}
	return append([]string{}, value...)
}
func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneGatewayConfig(value *model.GatewayConfig) *model.GatewayConfig {
	if value == nil {
		return nil
	}
	copy := *value
	copy.VersionBindings = append([]model.GatewayVersionBinding(nil), value.VersionBindings...)
	return &copy
}

// EffectiveServicePlanHash excludes identifiers, timestamps, and gateway
// configuration so it describes service configuration rather than persistence
// history or shared gateway state.
func EffectiveServicePlanHash(plan model.EffectiveServicePlan) (string, error) {
	type fingerprintComponent struct {
		Name          string
		Image         string
		Entrypoint    []string
		Command       []string
		Env           []model.VersionComponentEnv
		Mounts        []model.VersionComponentMount
		Dependencies  []model.VersionComponentDependency
		Healthcheck   *model.VersionComponentHealthcheck
		Resources     *model.VersionComponentResources
		PullPolicy    string
		RestartPolicy *string
		Tmpfs         []model.VersionComponentTmpfs
		Ulimits       []model.VersionComponentUlimit
		Devices       []model.VersionComponentDeviceRequest
		Endpoints     []model.VersionComponentEndpoint
	}
	type fingerprint struct {
		AppCode            string
		AppKind            string
		VersionLabel       string
		InstanceKey        string
		ServiceCode        string
		JoinTraefikNetwork bool
		Components         []fingerprintComponent
	}
	components := make([]fingerprintComponent, 0, len(plan.Components))
	for _, component := range plan.Components {
		components = append(components, fingerprintComponent{Name: component.Name, Image: component.Image, Entrypoint: component.Entrypoint, Command: component.Command, Env: component.Env, Mounts: component.Mounts, Dependencies: component.Dependencies, Healthcheck: component.Healthcheck, Resources: component.Resources, PullPolicy: component.PullPolicy, RestartPolicy: component.RestartPolicy, Tmpfs: component.Tmpfs, Ulimits: component.Ulimits, Devices: component.Devices, Endpoints: component.Endpoints})
	}
	sort.Slice(components, func(i, j int) bool { return components[i].Name < components[j].Name })
	data := fingerprint{AppCode: plan.Application.Code, AppKind: plan.Application.Kind, VersionLabel: plan.Version.Label, InstanceKey: plan.Service.InstanceKey, ServiceCode: plan.Service.Code, JoinTraefikNetwork: plan.JoinsTraefikNetwork(), Components: components}
	raw, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("encode effective service plan: %w", err)
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}
