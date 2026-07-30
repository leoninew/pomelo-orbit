package applicationhandler

import (
	"testing"

	applicationv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/application"
)

func TestVersionComponentCreateInputUsesBasicFieldsOnly(t *testing.T) {
	pullPolicy := "always"
	restartPolicy := "unless-stopped"
	input := versionComponentCreateInput(&applicationv1.VersionComponentCreateReq{
		Name:          "api",
		Image:         "nginx:1.27",
		Command:       "nginx -g 'daemon off;'",
		PullPolicy:    &pullPolicy,
		RestartPolicy: &restartPolicy,
	})

	if input.Name != "api" || input.Image != "nginx:1.27" || input.Command != "nginx -g 'daemon off;'" {
		t.Fatalf("unexpected basic input: %+v", input)
	}
	if input.PullPolicy != &pullPolicy || input.RestartPolicy != &restartPolicy {
		t.Fatalf("unexpected policies: %+v", input)
	}
	if len(input.Env) != 0 || len(input.Ports) != 0 || len(input.Mounts) != 0 || len(input.Dependencies) != 0 || input.Healthcheck != nil || input.Resources != nil || len(input.Tmpfs) != 0 || len(input.Ulimits) != 0 {
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
