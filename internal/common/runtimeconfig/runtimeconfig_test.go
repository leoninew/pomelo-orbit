package runtimeconfig

import (
	"reflect"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestValidateAndResolveRuntimeConfig(t *testing.T) {
	versionEnv := `[{"key":"REQUIRED","value":"${API_TOKEN}"},{"key":"DEFAULTED","value":"${TIMEOUT:-30}"}]`
	components := []model.VersionComponent{{EnvJSON: ptr(`[{"key":"COPY","value":"${API_TOKEN}"}]`)}}

	missing, extra, err := Validate(map[string]string{"UNUSED": "x"}, &versionEnv, components)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(missing, []string{"API_TOKEN"}) || !reflect.DeepEqual(extra, []string{"UNUSED"}) {
		t.Fatalf("unexpected validation result: missing=%v extra=%v", missing, extra)
	}

	resolved, extra, err := Resolve(map[string]string{"API_TOKEN": "token", "UNUSED": "x"}, &versionEnv, components)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resolved, map[string]string{"API_TOKEN": "token", "TIMEOUT": "30"}) {
		t.Fatalf("unexpected resolved config: %+v", resolved)
	}
	if !reflect.DeepEqual(extra, []string{"UNUSED"}) {
		t.Fatalf("unexpected extra config: %+v", extra)
	}
}

func ptr(value string) *string { return &value }
