package deploymentsvc

import (
	"encoding/json"
	"fmt"
	"strings"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const (
	containerVersionIDLabel    = "pomelo.orbit.version-id"
	containerVersionLabelLabel = "pomelo.orbit.version-label"
)

func injectContainerVersionLabels(services map[string]any, version model.Version) {
	labels := []string{
		containerVersionIDLabel + "=" + version.Id,
		containerVersionLabelLabel + "=" + version.Label,
	}
	for _, raw := range services {
		service, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		existing, _ := service["labels"].([]string)
		service["labels"] = append(existing, labels...)
	}
}

func parseContainerLabelOutput(raw string) (map[string]map[string]string, error) {
	result := make(map[string]map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("invalid container label record %q", line)
		}
		var labels map[string]string
		if err := json.Unmarshal([]byte(parts[1]), &labels); err != nil {
			return nil, fmt.Errorf("decode labels for container %s: %w", parts[0], err)
		}
		result[parts[0]] = labels
	}
	return result, nil
}

func applyContainerVersionLabels(containers []deploymentdto.RuntimeContainer, labelsByID map[string]map[string]string) {
	for index := range containers {
		labels, ok := containerLabelsForID(containers[index].ID, labelsByID)
		if !ok {
			continue
		}
		versionID := labels[containerVersionIDLabel]
		if versionID == "" {
			continue
		}
		containers[index].VersionID = versionID
		containers[index].VersionLabel = labels[containerVersionLabelLabel]
	}
}

func containerLabelsForID(id string, labelsByID map[string]map[string]string) (map[string]string, bool) {
	if labels, ok := labelsByID[id]; ok {
		return labels, true
	}
	for inspectedID, labels := range labelsByID {
		if strings.HasPrefix(inspectedID, id) {
			return labels, true
		}
	}
	return nil, false
}
