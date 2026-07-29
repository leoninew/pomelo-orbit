package deploymentsvc

import "testing"

func TestParseComposePsOutput(t *testing.T) {
	t.Parallel()

	raw := `[
  {"ID":"abc","Name":"app-web-1","Service":"web","State":"running","Status":"Up 2 hours","Health":"healthy","Image":"nginx:latest"},
  {"ID":"def","Name":"app-db-1","Service":"db","State":"running","Status":"Up About a minute","Health":"starting","Image":"postgres:16"}
]`
	containers, err := parseComposePsOutput(raw)
	if err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if len(containers) != 2 {
		t.Fatalf("container count: got %d want 2", len(containers))
	}
	if got, want := containers[0].Health, "healthy"; got != want {
		t.Fatalf("container0 Health: got %q want %q", got, want)
	}
	if got, want := containers[1].Health, "starting"; got != want {
		t.Fatalf("container1 Health: got %q want %q", got, want)
	}
	if got, want := containers[0].Status, "Up 2 hours"; got != want {
		t.Fatalf("container0 Status: got %q want %q", got, want)
	}
	if containers[0].Id != "abc" || containers[0].Name != "app-web-1" || containers[0].Service != "web" || containers[0].State != "running" || containers[0].Image != "nginx:latest" {
		t.Fatalf("container0: got %+v", containers[0])
	}
}

func TestParseComposePsOutputAllowsEmptyArray(t *testing.T) {
	t.Parallel()

	containers, err := parseComposePsOutput("[]")
	if err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if len(containers) != 0 {
		t.Fatalf("container count: got %d want 0", len(containers))
	}
}

func TestParseComposePsOutputAcceptsJSONRecords(t *testing.T) {
	t.Parallel()

	raw := "{\"ID\":\"abc\",\"Service\":\"web\",\"State\":\"running\",\"Status\":\"Up 1 minute (health: healthy)\",\"Health\":\"healthy\"}\n{\"ID\":\"def\",\"Service\":\"db\",\"State\":\"running\",\"Status\":\"Up 1 minute (health: starting)\",\"Health\":\"starting\"}"
	containers, err := parseComposePsOutput(raw)
	if err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if len(containers) != 2 {
		t.Fatalf("container count: got %d want 2", len(containers))
	}
	if containers[0].Id != "abc" || containers[0].Health != "healthy" || containers[1].Id != "def" || containers[1].Health != "starting" {
		t.Fatalf("unexpected containers: %+v", containers)
	}
}

func TestParseComposePsOutputRejectsNonJSON(t *testing.T) {
	t.Parallel()

	if _, err := parseComposePsOutput("not json at all"); err == nil {
		t.Fatal("expected non-JSON output to fail")
	}
}
