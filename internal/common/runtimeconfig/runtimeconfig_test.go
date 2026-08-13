package runtimeconfig

import (
	"reflect"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestValidateAndResolveRuntimeConfig(t *testing.T) {
	components := []model.VersionComponent{{Env: []model.VersionComponentEnv{
		{Key: "COPY", Value: "${API_TOKEN}"},
		{Key: "TIMEOUT", Value: "${TIMEOUT:-30}"},
	}}}

	missing, extra := Validate(map[string]string{"UNUSED": "x"}, components)
	if !reflect.DeepEqual(missing, []string{"API_TOKEN"}) || !reflect.DeepEqual(extra, []string{"UNUSED"}) {
		t.Fatalf("unexpected validation result: missing=%v extra=%v", missing, extra)
	}

	resolved, extra, err := Resolve(map[string]string{"API_TOKEN": "token", "UNUSED": "x"}, components)
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
