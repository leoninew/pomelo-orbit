package deploymentsvc

import (
	"context"
	"strings"
	"testing"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestRenderComposeIncludesStructuredRuntimeFields(t *testing.T) {
	restartPolicy := "unless-stopped"
	service := Service{}
	compose, err := service.RenderCompose(context.Background(), RenderInput{
		Plan: model.EffectiveServicePlan{
			Application: model.Application{Code: "demo", Kind: status.ApplicationKindStandard},
			Version:     model.Version{Id: "version-1"},
			Service:     model.Service{InstanceKey: "default"},
			Components: []model.EffectiveServiceComponent{{
				Name: "web", Image: "nginx", RestartPolicy: &restartPolicy,
				Tmpfs:   []model.VersionComponentTmpfs{{Target: "/tmp", SizeBytes: 1048576, Mode: "1777"}},
				Ulimits: []model.VersionComponentUlimit{{Name: "memlock", Soft: -1, Hard: -1}},
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"restart: unless-stopped", "/tmp:size=1048576,mode=1777", "memlock:", "soft: -1"} {
		if !strings.Contains(compose, expected) {
			t.Fatalf("compose missing %q:\n%s", expected, compose)
		}
	}
}

func TestRenderComponentHealthcheckUsesSingleCommandText(t *testing.T) {
	tests := []struct {
		name        string
		healthcheck model.VersionComponentHealthcheck
		want        []string
	}{
		{
			name: "command",
			healthcheck: model.VersionComponentHealthcheck{
				TestMode: "CMD",
				Test:     `curl -f 'http://localhost:8080/health check'`,
			},
			want: []string{"CMD", "curl", "-f", "http://localhost:8080/health check"},
		},
		{
			name: "shell command",
			healthcheck: model.VersionComponentHealthcheck{
				TestMode: "CMD-SHELL",
				Test:     `curl -f "http://localhost:8080/health check" || exit 1`,
			},
			want: []string{"CMD-SHELL", `curl -f "http://localhost:8080/health check" || exit 1`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			healthcheck, err := renderComponentHealthcheck(&test.healthcheck)
			if err != nil {
				t.Fatal(err)
			}
			got, ok := healthcheck["test"].([]string)
			if !ok {
				t.Fatalf("healthcheck test type = %T", healthcheck["test"])
			}
			if strings.Join(got, "\x00") != strings.Join(test.want, "\x00") {
				t.Fatalf("healthcheck test = %#v, want %#v", got, test.want)
			}
		})
	}
}
