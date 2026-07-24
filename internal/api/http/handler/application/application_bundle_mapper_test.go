package applicationhandler

import "testing"

func TestVersionUpdateInputFromJSONClearsEmptyExposes(t *testing.T) {
	body := []byte(`{
		"label":"v1",
		"components":[{"name":"traefik","image":"traefik","ports_json":"[\"80:80\"]"}],
		"exposes":[]
	}`)
	input, err := versionUpdateInputFromJSON(body)
	if err != nil {
		t.Fatal(err)
	}
	if input.Exposes == nil {
		t.Fatal("expected exposes pointer set so empty list replaces existing rows")
	}
	if len(*input.Exposes) != 0 {
		t.Fatalf("expected empty exposes, got %d", len(*input.Exposes))
	}
	if input.Components == nil || len(*input.Components) != 1 {
		t.Fatalf("expected one component, got %#v", input.Components)
	}
}

func TestVersionUpdateInputFromJSONOmitsMissingExposes(t *testing.T) {
	body := []byte(`{"label":"v2"}`)
	input, err := versionUpdateInputFromJSON(body)
	if err != nil {
		t.Fatal(err)
	}
	if input.Exposes != nil {
		t.Fatal("missing exposes key must leave collection unchanged (nil)")
	}
	if input.Components != nil {
		t.Fatal("missing components key must leave collection unchanged (nil)")
	}
	if input.Label == nil || *input.Label != "v2" {
		t.Fatalf("expected label v2, got %#v", input.Label)
	}
}
