package deploymentsvc

import (
	"encoding/json"
	"path"
	"sort"
	"strings"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

const sharedMountsLabel = "io.pomelo-orbit.shared-mounts"

func renderSharedMountsLabel(mounts []model.VersionComponentMount) (string, error) {
	var targets []string
	for _, mount := range mounts {
		if mount.Shared {
			targets = append(targets, mount.Target)
		}
	}
	if len(targets) == 0 {
		return "", nil
	}
	sort.Strings(targets)
	encoded, err := json.Marshal(targets)
	return string(encoded), err
}

func sharedMountTargets(labels map[string]string) (map[string]bool, error) {
	result := map[string]bool{}
	if value := labels[sharedMountsLabel]; value != "" {
		var targets []string
		if err := json.Unmarshal([]byte(value), &targets); err != nil {
			return nil, err
		}
		for _, target := range targets {
			result[sharedMountTargetKey(target)] = true
		}
	}
	return result, nil
}

func sharedMountTargetKey(value string) string {
	return path.Clean(dockerMountComparisonPath(value, ""))
}

func mountSourceContains(parent, child, platform string) bool {
	parent = strings.TrimRight(path.Clean(dockerMountComparisonPath(parent, platform)), "/")
	child = strings.TrimRight(path.Clean(dockerMountComparisonPath(child, platform)), "/")
	return parent == child || strings.HasPrefix(child, parent+"/")
}
