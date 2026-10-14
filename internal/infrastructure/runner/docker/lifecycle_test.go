package dockerrunner

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

type lifecycleRunner struct {
	command  string
	target   environmentport.Target
	location deploymentport.ServiceLocation
	ctx      context.Context
}

func (r *lifecycleRunner) Run(ctx context.Context, target environmentport.Target, location deploymentport.ServiceLocation, output io.Writer, name string, args ...string) error {
	r.ctx, r.target, r.location = ctx, target, location
	r.command = name + " " + strings.Join(args, " ")
	if _, err := io.WriteString(output, "operation output"); err != nil {
		return err
	}
	return ctx.Err()
}

func TestLifecycleCommandsMatchExecutionAndPreserveContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	scope := deploymentport.RuntimeScope{Project: "service-a", Location: deploymentport.ServiceLocation{Code: "service-a", Directory: "/custom/service"}}
	scope.Target.Environment.Id = "environment-a"
	for _, tc := range []struct {
		name         string
		stop, option bool
		want         string
	}{
		{"deploy", false, false, "docker compose -p service-a -f docker-compose.yml up -d --remove-orphans --pull missing"},
		{"recreate", false, true, "docker compose -p service-a -f docker-compose.yml up -d --remove-orphans --pull missing --force-recreate"},
		{"stop", true, false, "docker compose -p service-a -f docker-compose.yml down"},
		{"remove volumes", true, true, "docker compose -p service-a -f docker-compose.yml down -v"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &lifecycleRunner{}
			lifecycle := NewLifecycle(runner)
			var log bytes.Buffer
			var description string
			var err error
			if tc.stop {
				description = (Commands{}).StopDescription(scope.Project, tc.option)
				err = lifecycle.Stop(ctx, scope, tc.option, &log)
			} else {
				settings := deploymentport.DeploySettings{PullPolicy: "missing", ForceRecreate: tc.option}
				description = (Commands{}).DeployDescription(scope.Project, settings)
				err = lifecycle.Deploy(ctx, scope, settings, &log)
			}
			if description != tc.want || runner.command != description {
				t.Fatalf("description=%q executed=%q", description, runner.command)
			}
			if !errors.Is(err, context.Canceled) || runner.ctx != ctx || runner.target.Environment.Id != scope.Target.Environment.Id || runner.location != scope.Location || log.String() != "operation output" {
				t.Fatalf("error=%v runner=%+v log=%q", err, runner, log.String())
			}
		})
	}
}
