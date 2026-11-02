package applicationsvc

import (
	"strings"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestValidateControlledFileRejectsReadOnly(t *testing.T) {
	mount := model.VersionComponentMount{SourceType: "controlled_file", Source: "./config", Target: "/config", Mode: "0644", ReadOnly: true}
	if err := validateComponentMount(mount); err == nil {
		t.Fatal("controlled file must not accept read_only")
	}
	mount.SourceType = "directory"
	mount.Mode = ""
	if err := validateComponentMount(mount); err != nil {
		t.Fatalf("directory must support read_only: %v", err)
	}
}

func TestValidateComponentRuntimeFieldsRestartPolicy(t *testing.T) {
	for _, policy := range []string{"no", "on-failure", "always", "unless-stopped"} {
		policy := policy
		t.Run(policy, func(t *testing.T) {
			if err := validateComponentRuntimeFields(model.VersionComponent{RestartPolicy: &policy}); err != nil {
				t.Fatalf("validateComponentRuntimeFields() error = %v", err)
			}
		})
	}
	for _, policy := range []string{"", "on-demand"} {
		policy := policy
		t.Run("invalid_"+policy, func(t *testing.T) {
			if err := validateComponentRuntimeFields(model.VersionComponent{RestartPolicy: &policy}); err == nil {
				t.Fatal("validateComponentRuntimeFields() accepted invalid policy")
			}
		})
	}
	if err := validateComponentRuntimeFields(model.VersionComponent{}); err == nil || !strings.Contains(err.Error(), "restart_policy is required") {
		t.Fatalf("nil restart policy error = %v", err)
	}
}
