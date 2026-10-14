package deploymentsvc

import (
	"reflect"
	"testing"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
)

func TestRuntimeConfigurationDifferencesUseServiceAndImageFacts(t *testing.T) {
	expected := map[string]string{"api": "api:v1", "worker": "worker:v1"}
	for _, tc := range []struct {
		name       string
		actual     map[string]string
		containers []deploymentdto.RuntimeContainer
		want       []string
	}{
		{"consistent", expected, []deploymentdto.RuntimeContainer{{Service: "api"}, {Service: "worker"}}, []string{}},
		{"missing service", map[string]string{"api": "api:v1"}, []deploymentdto.RuntimeContainer{{Service: "api"}}, []string{"Compose config is missing expected service worker"}},
		{"missing container", expected, []deploymentdto.RuntimeContainer{{Service: "api"}}, []string{"Docker has no running container for expected service worker"}},
		{"changed or missing image", map[string]string{"api": "api:v2", "worker": ""}, []deploymentdto.RuntimeContainer{{Service: "api"}, {Service: "worker"}}, []string{"service api image differs between preview and Compose config", "service worker image differs between preview and Compose config"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := compareRuntimeServices(expected, tc.actual, tc.containers); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("differences=%v want=%v", got, tc.want)
			}
		})
	}
}
