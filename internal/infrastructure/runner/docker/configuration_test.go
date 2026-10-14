package dockerrunner

import (
	"reflect"
	"testing"
)

func TestConfigurationServicesDecodesFactsAndRejectsInvalidDocuments(t *testing.T) {
	for _, tc := range []struct {
		name, content string
		want          map[string]string
		wantError     bool
	}{
		{"services", "services:\n  web:\n    image: nginx:latest\n    ports: [\"80:80\"]\n  worker: {}", map[string]string{"web": "nginx:latest", "worker": ""}, false},
		{"empty services", "services: {}", map[string]string{}, false},
		{"missing services", "networks: {}", nil, true},
		{"invalid yaml", "services: [", nil, true},
		{"invalid shape", "services: text", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := (RuntimeInspector{}).ConfigurationServices(tc.content)
			if (err != nil) != tc.wantError || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("services=%v err=%v", got, err)
			}
		})
	}
}
