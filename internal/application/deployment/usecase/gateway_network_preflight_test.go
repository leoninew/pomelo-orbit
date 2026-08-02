package deploymentsvc

import (
	"context"
	"errors"
	"strings"
	"testing"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

type gatewayNetworkQueryRunner struct {
	output string
	err    error
	called bool
}

func (r *gatewayNetworkQueryRunner) Run(context.Context, string, string, ...string) (string, error) {
	r.called = true
	return r.output, r.err
}

func TestPreflightGatewayNetworkAllowsInitialNetworkCreation(t *testing.T) {
	runner := &gatewayNetworkQueryRunner{output: "Error response from daemon: network traefik not found\n", err: errors.New("exit status 1")}
	err := preflightGatewayNetwork(context.Background(), runner, model.Application{Code: "traefik", Kind: status.ApplicationKindGateway})
	if err != nil {
		t.Fatalf("preflight error = %v", err)
	}
	if !runner.called {
		t.Fatal("network inspection was not called")
	}
}

func TestPreflightGatewayNetworkRejectsIncompatibleComposeLabels(t *testing.T) {
	runner := &gatewayNetworkQueryRunner{output: `[{"Name":"traefik","Labels":{"com.docker.compose.network":"default"}}]`}
	err := preflightGatewayNetwork(context.Background(), runner, model.Application{Code: "traefik", Kind: status.ApplicationKindGateway})
	if err == nil {
		t.Fatal("expected preflight conflict")
	}
	if !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("preflight error = %v", err)
	}
	if !strings.Contains(err.Error(), "Remove the existing traefik network") {
		t.Fatalf("preflight error = %v", err)
	}
}

func TestPreflightGatewayNetworkAllowsMatchingNetwork(t *testing.T) {
	runner := &gatewayNetworkQueryRunner{output: `[{"Name":"traefik","Labels":{"com.docker.compose.network":"traefik"}}]`}
	err := preflightGatewayNetwork(context.Background(), runner, model.Application{Code: "traefik", Kind: status.ApplicationKindGateway})
	if err != nil {
		t.Fatalf("preflight error = %v", err)
	}
}

func TestPreflightGatewayNetworkSkipsStandardApplication(t *testing.T) {
	runner := &gatewayNetworkQueryRunner{}
	err := preflightGatewayNetwork(context.Background(), runner, model.Application{Code: "app", Kind: status.ApplicationKindStandard})
	if err != nil {
		t.Fatalf("preflight error = %v", err)
	}
	if runner.called {
		t.Fatal("standard application must not inspect the gateway network")
	}
}
