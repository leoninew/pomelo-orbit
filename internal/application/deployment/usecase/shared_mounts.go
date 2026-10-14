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

func mountSourceContains(parent, child, platform string) bool {
	parent = strings.TrimRight(path.Clean(dockerMountComparisonPath(parent, platform)), "/")
	child = strings.TrimRight(path.Clean(dockerMountComparisonPath(child, platform)), "/")
	return parent == child || strings.HasPrefix(child, parent+"/")
}
