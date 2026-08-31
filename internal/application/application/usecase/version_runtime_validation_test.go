package applicationsvc

import (
	"strings"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

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
