package dockerrunner

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

type logRunner struct {
	output  string
	err     error
	command []string
	target  environmentport.Target
}

func (r *logRunner) QueryAtEnvironmentRoot(_ context.Context, target environmentport.Target, name string, args ...string) (string, error) {
	r.command, r.target = append([]string{name}, args...), target
	return r.output, r.err
}

func (r *logRunner) StreamAtEnvironmentRoot(_ context.Context, target environmentport.Target, output io.Writer, name string, args ...string) error {
	r.command, r.target = append([]string{name}, args...), target
	if _, err := io.WriteString(output, r.output); err != nil {
		return err
	}
	return r.err
}

func TestContainerLogSourcesUseProjectAndComponentLabels(t *testing.T) {
	failure := errors.New("daemon unavailable")
	for _, tc := range []struct {
		name, output string
		failure      error
		want         []string
		wantError    bool
	}{
		{"containers", "{\"ID\":\"one\"}\n{\"ID\":\"two\"}", nil, []string{"one", "two"}, false},
		{"empty", "", nil, []string{}, false},
		{"query failure", "", failure, nil, true},
		{"invalid observation", "not JSON", nil, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &logRunner{output: tc.output, err: tc.failure}
			ids, err := NewContainerLogs(runner).ContainerIds(context.Background(), environmentport.Target{}, "service-a", "web")
			if !reflect.DeepEqual(ids, tc.want) || (err != nil) != tc.wantError {
				t.Fatalf("ids=%v err=%v", ids, err)
			}
			if tc.failure != nil && !errors.Is(err, tc.failure) {
				t.Fatalf("lost query error: %v", err)
			}
			want := []string{"docker", "ps", "--all", "--filter", "label=com.docker.compose.project=service-a", "--format", "{{json .}}", "--filter", "label=com.docker.compose.service=web"}
			if !reflect.DeepEqual(runner.command, want) {
				t.Fatalf("command=%v", runner.command)
			}
		})
	}
}

func TestContainerLogFollowDoesNotRequireComposeWorkspace(t *testing.T) {
	for _, request := range []deploymentport.ContainerLogRequest{
		{Project: "service-a", Component: "web", Tail: 200},
		{Project: "service-a", Since: "2026-09-30T00:00:00Z"},
	} {
		t.Run(request.Since+request.Component, func(t *testing.T) {
			runner := &logRunner{output: "web | 2026-09-30T00:00:00Z hello\n", err: context.Canceled}
			var output bytes.Buffer
			target := environmentport.Target{}
			target.Environment.Id = "environment-a"
			err := NewContainerLogs(runner).Follow(context.Background(), target, request, &output)
			if !errors.Is(err, context.Canceled) || output.String() != runner.output || runner.target.Environment.Id != target.Environment.Id {
				t.Fatalf("output=%q target=%+v err=%v", output.String(), runner.target, err)
			}
			want := []string{"docker", "compose", "-p", "service-a", "logs", "--follow", "--timestamps", "--no-color"}
			if request.Since != "" {
				want = append(want, "--since", request.Since)
			} else {
				want = append(want, "--tail", "200", "web")
			}
			if !reflect.DeepEqual(runner.command, want) {
				t.Fatalf("command=%v", runner.command)
			}
		})
	}
}
