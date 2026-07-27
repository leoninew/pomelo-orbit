package deploymentsvc

import "testing"

func TestParseDependsOnJSONPreservesHealthConditions(t *testing.T) {
	raw := `{"mysql":{"condition":"service_healthy"},"redis":{"condition":"service_started"}}`
	depends, names, err := parseDependsOnJSON(&raw)
	if err != nil {
		t.Fatalf("parse depends_on: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("dependency names = %#v", names)
	}
	configured, ok := depends.(map[string]map[string]string)
	if !ok {
		t.Fatalf("depends type = %T", depends)
	}
	if configured["mysql"]["condition"] != "service_healthy" {
		t.Fatalf("mysql condition = %#v", configured["mysql"])
	}
}
