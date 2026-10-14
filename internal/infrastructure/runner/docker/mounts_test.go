package dockerrunner

import (
	"context"
	"errors"
	"reflect"
	"testing"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

type mountQuery struct {
	outputs  []string
	errors   []error
	commands [][]string
}

func (q *mountQuery) QueryAtEnvironmentRoot(_ context.Context, _ environmentport.Target, name string, args ...string) (string, error) {
	index := len(q.commands)
	q.commands = append(q.commands, append([]string{name}, args...))
	return q.outputs[index], q.errors[index]
}

func TestRunningBindMountsReadsLiveMountsAndSharedTargets(t *testing.T) {
	query := &mountQuery{outputs: []string{"one\ntwo\n", `[
 {"State":{"Running":true},"Config":{"Labels":{"io.pomelo-orbit.shared-mounts":"[\"/data/\"]"}},"Mounts":[
  {"Type":"bind","Source":"/host","Destination":"/data"},
  {"Type":"bind","Source":"/private","Destination":"/data/private"},
  {"Type":"volume","Source":"named","Destination":"/data"}]},
 {"State":{"Running":false},"Mounts":[{"Type":"bind","Source":"/stopped"}]}
]`}, errors: []error{nil, nil}}
	mounts, err := NewInventory(query).RunningBindMounts(context.Background(), environmentport.Target{}, "service-a")
	want := []deploymentport.RunningBindMount{{Source: "/host", Shared: true}, {Source: "/private"}}
	if err != nil || !reflect.DeepEqual(mounts, want) {
		t.Fatalf("mounts=%+v, err=%v", mounts, err)
	}
	commands := [][]string{
		{"docker", "ps", "--filter", "label=com.docker.compose.project=service-a", "--filter", "status=running", "--format", "{{.ID}}"},
		{"docker", "inspect", "--type", "container", "one", "two"},
	}
	if !reflect.DeepEqual(query.commands, commands) {
		t.Fatalf("commands=%v", query.commands)
	}
}

func TestRunningBindMountsHandlesEmptyListAndInspectionFailures(t *testing.T) {
	failure := errors.New("daemon unavailable")
	for _, tc := range []struct {
		name      string
		outputs   []string
		errors    []error
		wantError bool
	}{
		{"empty", []string{""}, []error{nil}, false},
		{"list failure", []string{""}, []error{failure}, true},
		{"inspect failure", []string{"one", ""}, []error{nil, failure}, true},
		{"invalid observation", []string{"one", "invalid JSON"}, []error{nil, nil}, true},
		{"invalid sharing declaration", []string{"one", `[{"State":{"Running":true},"Config":{"Labels":{"io.pomelo-orbit.shared-mounts":"invalid"}}}]`}, []error{nil, nil}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			query := &mountQuery{outputs: tc.outputs, errors: tc.errors}
			mounts, err := NewInventory(query).RunningBindMounts(context.Background(), environmentport.Target{}, "service-a")
			if (err != nil) != tc.wantError || len(mounts) != 0 || len(query.commands) != len(tc.outputs) {
				t.Fatalf("mounts=%v, err=%v, commands=%v", mounts, err, query.commands)
			}
			if tc.errors[len(tc.errors)-1] != nil && !errors.Is(err, failure) {
				t.Fatalf("command failure lost: %v", err)
			}
		})
	}
}
