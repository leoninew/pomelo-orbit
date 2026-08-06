package dialogueusecase

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// turnExecution enforces the minimum safe order for a single user turn. It
// does not infer a desired configuration from prose; it only prevents known
// unsafe transitions after the model has selected a concrete MCP write.
type turnExecution struct {
	versionsNeedingRead  map[string]struct{}
	servicesNeedingCheck map[string]struct{}
	deployedServices     map[string]struct{}
	unwaitedDeployments  map[string]struct{}
}

func newTurnExecution() *turnExecution {
	return &turnExecution{
		versionsNeedingRead:  make(map[string]struct{}),
		servicesNeedingCheck: make(map[string]struct{}),
		deployedServices:     make(map[string]struct{}),
		unwaitedDeployments:  make(map[string]struct{}),
	}
}

func (e *turnExecution) before(name string, arguments json.RawMessage) error {
	values := toolArguments(arguments)
	switch name {
	case "orbit_deploy":
		serviceID := stringArgument(values, "service_id")
		if serviceID == "" {
			return fmt.Errorf("orbit_deploy requires service_id")
		}
		if _, exists := e.deployedServices[serviceID]; exists {
			return fmt.Errorf("orbit_deploy is limited to one call per service in a dialogue turn; reuse the existing deployment result")
		}
		if versions := e.pendingVersionIds(); len(versions) > 0 {
			return fmt.Errorf("read updated Version state with orbit_get_version before deployment: %s", strings.Join(versions, ", "))
		}
		if _, exists := e.servicesNeedingCheck[serviceID]; exists {
			return fmt.Errorf("preview the updated Service with orbit_preview_service before deployment: %s", serviceID)
		}
	}
	return nil
}

func (e *turnExecution) record(name string, arguments, result json.RawMessage, isError bool) {
	if isError {
		return
	}
	values := toolArguments(arguments)
	switch {
	case name == "orbit_get_version":
		delete(e.versionsNeedingRead, stringArgument(values, "version_id"))
	case isVersionComponentWrite(name):
		if versionID := stringArgument(values, "version_id"); versionID != "" {
			e.versionsNeedingRead[versionID] = struct{}{}
		}
	case isServiceWrite(name):
		if serviceID := stringArgument(values, "service_id"); serviceID != "" {
			e.servicesNeedingCheck[serviceID] = struct{}{}
		}
	case name == "orbit_create_service":
		if serviceID := resultResourceID(result, "service_id"); serviceID != "" {
			e.servicesNeedingCheck[serviceID] = struct{}{}
		}
	case name == "orbit_preview_service":
		delete(e.servicesNeedingCheck, stringArgument(values, "service_id"))
	case name == "orbit_deploy":
		serviceID := stringArgument(values, "service_id")
		if serviceID != "" {
			e.deployedServices[serviceID] = struct{}{}
		}
		if deploymentID := resultResourceID(result, "deployment_id"); deploymentID != "" {
			e.unwaitedDeployments[deploymentID] = struct{}{}
		}
	case name == "orbit_wait_deployment":
		delete(e.unwaitedDeployments, stringArgument(values, "deployment_id"))
	}
}

func isVersionComponentWrite(name string) bool {
	return name == "orbit_create_version_component" || strings.HasPrefix(name, "orbit_update_version_component_")
}

func isServiceWrite(name string) bool {
	return name == "orbit_update_service_component_overlay" || name == "orbit_update_service_env" || name == "orbit_update_service_basic"
}

func (e *turnExecution) pendingVersionIds() []string {
	ids := make([]string, 0, len(e.versionsNeedingRead))
	for id := range e.versionsNeedingRead {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (e *turnExecution) unwaitedDeploymentIds() []string {
	ids := make([]string, 0, len(e.unwaitedDeployments))
	for id := range e.unwaitedDeployments {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func toolArguments(arguments json.RawMessage) map[string]any {
	values := map[string]any{}
	_ = json.Unmarshal(arguments, &values)
	return values
}

func stringArgument(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func resultResourceID(result json.RawMessage, key string) string {
	var payload struct {
		ResourceIDs map[string]string `json:"resource_ids"`
	}
	if json.Unmarshal(result, &payload) != nil {
		return ""
	}
	return strings.TrimSpace(payload.ResourceIDs[key])
}
