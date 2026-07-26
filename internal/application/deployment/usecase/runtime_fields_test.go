package deploymentsvc

import (
	"context"
	"strings"
	"testing"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestRenderComposeIncludesStructuredRuntimeFieldsAndSecretEnvFile(t *testing.T) {
	restartPolicy := "unless-stopped"
	tmpfsJSON := `[{"target":"/tmp","size_bytes":1048576,"mode":"1777"}]`
	ulimitsJSON := `[{"name":"memlock","soft":-1,"hard":-1}]`
	service := Service{}
	compose, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "demo", Kind: status.ApplicationKindStandard},
		Version: model.Version{Id: "version-1"},
		Service: model.Service{InstanceKey: "default"},
		Components: []model.VersionComponent{{
			Name: "web", Image: "nginx", RestartPolicy: &restartPolicy, TmpfsJSON: &tmpfsJSON, UlimitsJSON: &ulimitsJSON,
			SecretEnvRefs: []model.VersionComponentSecretEnvRef{{EnvKey: "MYSQL_PASSWORD", CredentialId: "credential-1", DataKey: "MYSQL_PASSWORD"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"restart: unless-stopped", "env_file:", ".runtime/web.env", "/tmp:size=1048576,mode=1777", "memlock:", "soft: -1"} {
		if !strings.Contains(compose, expected) {
			t.Fatalf("compose missing %q:\n%s", expected, compose)
		}
	}
	if strings.Contains(compose, "runtime-test-secret") {
		t.Fatalf("compose must never contain runtime credential data:\n%s", compose)
	}
}

func TestSecretRedactorMasksValuesAndEnvironmentAssignments(t *testing.T) {
	redactor := NewSecretRedactor(map[string]string{"MYSQL_PASSWORD": "runtime-test-secret"})
	value := redactor.RedactText("password=runtime-test-secret MYSQL_PASSWORD=another-value")
	if strings.Contains(value, "runtime-test-secret") || strings.Contains(value, "another-value") {
		t.Fatalf("secret redaction failed: %q", value)
	}
	if !strings.Contains(value, "[REDACTED]") {
		t.Fatalf("expected redacted marker: %q", value)
	}
}
