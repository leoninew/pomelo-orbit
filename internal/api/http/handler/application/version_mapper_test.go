package applicationhandler

import (
	"testing"

	applicationv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/application"
)

func TestVersionComponentCreateInputUsesBasicFieldsOnly(t *testing.T) {
	restartPolicy := "unless-stopped"
	input := versionComponentCreateInput(&applicationv1.VersionComponentCreateReq{
		Name:          "api",
		Image:         "nginx:1.27",
		Entrypoint:    "/docker-entrypoint.sh",
		Command:       "nginx -g 'daemon off;'",
		PullPolicy:    "always",
		RestartPolicy: &restartPolicy,
	})

	if input.Name != "api" || input.Image != "nginx:1.27" || input.Entrypoint != "/docker-entrypoint.sh" || input.Command != "nginx -g 'daemon off;'" {
		t.Fatalf("unexpected basic input: %+v", input)
	}
	if input.PullPolicy != "always" || input.RestartPolicy != &restartPolicy {
		t.Fatalf("unexpected policies: %+v", input)
	}
	if len(input.Env) != 0 || len(input.Endpoints) != 0 || len(input.Mounts) != 0 || len(input.Dependencies) != 0 || input.Healthcheck != nil || input.Resources != nil || len(input.Tmpfs) != 0 || len(input.Ulimits) != 0 {
		t.Fatalf("create input must not include non-basic configuration: %+v", input)
	}
}

func TestVersionUpdateInputMapsMetadata(t *testing.T) {
	label := "v2"
	note := "updated"
	input := versionUpdateInput(&applicationv1.VersionUpdateReq{Label: &label, Note: &note})
	if input.Label == nil || *input.Label != "v2" {
		t.Fatalf("expected label v2, got %#v", input.Label)
	}
	if input.Note == nil || *input.Note != note {
		t.Fatalf("expected note %q, got %#v", note, input.Note)
	}
}
