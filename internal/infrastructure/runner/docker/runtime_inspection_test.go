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

type runtimeQuery struct {
	output   string
	err      error
	args     []string
	target   environmentport.Target
	location deploymentport.ServiceLocation
}

func (q *runtimeQuery) Query(_ context.Context, target environmentport.Target, location deploymentport.ServiceLocation, name string, args ...string) (string, error) {
	q.args, q.target, q.location = append([]string{name}, args...), target, location
	return q.output, q.err
}

func TestRuntimeQueriesPreserveScopeAndProtocol(t *testing.T) {
	ctx := context.Background()
	scope := deploymentport.RuntimeScope{Location: deploymentport.ServiceLocation{Code: "service-a", Directory: "/custom/runtime"}, Project: "service-a"}
	scope.Target.Environment.Id = "environment-a"
	for _, tc := range []struct {
		name string
		call func(RuntimeInspector) (string, error)
		args []string
		want string
	}{
		{"config", func(r RuntimeInspector) (string, error) { return r.Config(ctx, scope) }, []string{"compose", "-f", "docker-compose.yml", "config"}, " output\n"},
		{"tail logs", func(r RuntimeInspector) (string, error) { return r.Logs(ctx, scope, 20, "", []string{"web"}) }, []string{"compose", "-p", "service-a", "-f", "docker-compose.yml", "logs", "--tail", "20", "web"}, " output\n"},
		{"since logs", func(r RuntimeInspector) (string, error) { return r.Logs(ctx, scope, 20, "2026-10-14T00:00:00Z", nil) }, []string{"compose", "-p", "service-a", "-f", "docker-compose.yml", "logs", "--since", "2026-10-14T00:00:00Z"}, " output\n"},
		{"probe", func(r RuntimeInspector) (string, error) {
			return r.HTTPProbe(ctx, scope, "web", 8080, "/health?full=1")
		}, []string{"compose", "-p", "service-a", "-f", "docker-compose.yml", "exec", "-T", "web", "curl", "-fsS", "--max-time", "10", "http://127.0.0.1:8080/health?full=1"}, " output\n"},
		{"version", func(r RuntimeInspector) (string, error) { return r.Version(ctx, scope) }, []string{"version", "--format", "json"}, "output"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			query := &runtimeQuery{output: " output\n"}
			result, err := tc.call(NewRuntimeInspector(query))
			if err != nil || result != tc.want || !reflect.DeepEqual(query.args, append([]string{"docker"}, tc.args...)) || query.location != scope.Location || query.target.Environment.Id != "environment-a" {
				t.Fatalf("result=%q err=%v query=%+v", result, err, query)
			}
		})
	}
}

func TestContainerObservationPreservesDiagnosticShape(t *testing.T) {
	for _, output := range []string{`[{"ID":"one","State":"running"}]`, "{\"ID\":\"one\"}\n{\"ID\":\"two\"}", ""} {
		query := &runtimeQuery{output: output}
		result, err := NewRuntimeInspector(query).Containers(context.Background(), deploymentport.RuntimeScope{Project: "service-a"})
		if err != nil {
			t.Fatal(err)
		}
		switch {
		case output == "":
			if result.Raw != nil || len(result.Containers) != 0 {
				t.Fatalf("empty observation=%+v", result)
			}
		case strings.HasPrefix(output, "["):
			if _, ok := result.Raw.([]any); !ok || len(result.Containers) != 1 || result.Containers[0].State != "running" {
				t.Fatalf("array observation=%+v", result)
			}
		default:
			if result.Raw != output || len(result.Containers) != 2 {
				t.Fatalf("record observation=%+v", result)
			}
		}
		want := []string{"docker", "compose", "-p", "service-a", "-f", "docker-compose.yml", "ps", "--format", "json"}
		if !reflect.DeepEqual(query.args, want) {
			t.Fatalf("command=%v", query.args)
		}
	}
}

func TestRuntimeInspectionReturnsFactsAndRawEvidence(t *testing.T) {
	query := &runtimeQuery{output: `[{"Id":"one","RestartCount":3,"State":{"Status":"running","Health":{"Status":"healthy"}},"NetworkSettings":{"Networks":{"service-a_backend":{}}},"Extra":"retained"}]`}
	inspector := NewRuntimeInspector(query)
	result, err := inspector.InspectContainer(context.Background(), deploymentport.RuntimeScope{}, "one")
	if err != nil || result.State != "running" || result.Health != "healthy" || result.RestartCount != 3 || !reflect.DeepEqual(result.Networks, []string{"service-a_backend"}) {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if result.Raw.([]any)[0].(map[string]any)["Extra"] != "retained" || !reflect.DeepEqual(query.args, []string{"docker", "inspect", "one"}) {
		t.Fatalf("raw=%v command=%v", result.Raw, query.args)
	}
	query.output = `[{"Name":"service-a_backend"}]`
	network, err := inspector.InspectNetwork(context.Background(), deploymentport.RuntimeScope{}, "service-a_backend")
	if err != nil || network.([]any)[0].(map[string]any)["Name"] != "service-a_backend" || !reflect.DeepEqual(query.args, []string{"docker", "network", "inspect", "service-a_backend"}) {
		t.Fatalf("network=%v err=%v command=%v", network, err, query.args)
	}
}

func TestRuntimeInspectionPropagatesCommandAndDecodeFailures(t *testing.T) {
	ctx := context.Background()
	for _, call := range []func(RuntimeInspector) error{
		func(r RuntimeInspector) error { _, err := r.Containers(ctx, deploymentport.RuntimeScope{}); return err },
		func(r RuntimeInspector) error {
			_, err := r.InspectContainer(ctx, deploymentport.RuntimeScope{}, "one")
			return err
		},
		func(r RuntimeInspector) error {
			_, err := r.InspectNetwork(ctx, deploymentport.RuntimeScope{}, "backend")
			return err
		},
	} {
		query := &runtimeQuery{output: "daemon unavailable", err: errors.New("exit status 1")}
		if err := call(NewRuntimeInspector(query)); err == nil || !strings.Contains(err.Error(), query.output) {
			t.Fatalf("command error=%v", err)
		}
		query.err, query.output = nil, "not JSON"
		if err := call(NewRuntimeInspector(query)); err == nil {
			t.Fatal("invalid observation was accepted")
		}
	}
}
