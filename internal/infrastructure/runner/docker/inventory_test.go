package dockerrunner

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

type inventoryQuery struct {
	output  string
	err     error
	command []string
}

func (q *inventoryQuery) QueryAtEnvironmentRoot(_ context.Context, _ environmentport.Target, name string, args ...string) (string, error) {
	q.command = append([]string{name}, args...)
	return q.output, q.err
}

func TestInventoryReadsComposeProjectsAndExcludesOneoffContainers(t *testing.T) {
	query := &inventoryQuery{output: `
{"Labels":"com.docker.compose.project=example,com.docker.compose.oneoff=False","State":"running","Status":"Up 1 minute (unhealthy)"}
{"Labels":"com.docker.compose.project=example-extra","State":"exited"}
{"Labels":"com.docker.compose.project=example,com.docker.compose.oneoff=True","State":"running"}
{"Labels":"unmanaged=true","State":"running"}
`}
	got, err := NewInventory(query).Containers(context.Background(), environmentport.Target{})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]deploymentport.ContainerState{"example": {{State: "running", Unhealthy: true}}, "example-extra": {{State: "exited"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("inventory=%+v", got)
	}
	if !reflect.DeepEqual(query.command, []string{"docker", "ps", "--all", "--filter", "label=com.docker.compose.project", "--format", `{{json .}}`}) {
		t.Fatalf("command=%v", query.command)
	}
}

func TestInventoryPreservesCommandFailureAndRejectsInvalidOutput(t *testing.T) {
	failure := errors.New("exit status 1")
	query := &inventoryQuery{output: "daemon unavailable", err: failure}
	if _, err := NewInventory(query).Containers(context.Background(), environmentport.Target{}); !errors.Is(err, failure) || !strings.Contains(err.Error(), query.output) {
		t.Fatalf("error=%v", err)
	}
	query.output, query.err = "invalid JSON", nil
	if _, err := NewInventory(query).Containers(context.Background(), environmentport.Target{}); err == nil {
		t.Fatal("invalid output accepted")
	}
}
