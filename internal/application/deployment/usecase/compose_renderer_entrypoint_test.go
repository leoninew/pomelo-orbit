package deploymentsvc

import (
	"context"
	"strings"
	"testing"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestRenderComposeIncludesComponentEntrypoint(t *testing.T) {
	compose, err := Service{}.RenderCompose(context.Background(), RenderInput{
		Plan: model.EffectiveServicePlan{
			Application: model.Application{Code: "demo", Kind: status.ApplicationKindStandard},
			Version:     model.Version{Id: "version-1"},
			Components: []model.EffectiveServiceComponent{{
				Name: "api", Image: "example/api:latest",
				Entrypoint: []string{"/app/entrypoint", "serve"},
				Command:    []string{"--port", "8080"},
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(compose, "entrypoint:") || !strings.Contains(compose, "- /app/entrypoint") || !strings.Contains(compose, "- serve") {
		t.Fatalf("compose missing component entrypoint:\n%s", compose)
	}
}
