package gatewaysvc

import (
	"context"
	"strings"
	"testing"

	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	gatewayport "github.com/leoninew/pomelo-orbit/internal/application/gateway/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type createProjectFacts struct{}

func (createProjectFacts) Project(_ context.Context, id string) (model.Project, error) {
	return model.Project{Id: id}, nil
}
func (createProjectFacts) IsProjectMember(context.Context, string, string) (bool, error) {
	return true, nil
}

type createEnvironmentFacts struct {
	gatewayport.EnvironmentStore
	environment model.Environment
}

func (f createEnvironmentFacts) EnvironmentByProject(context.Context, string) (model.Environment, error) {
	return f.environment, nil
}

type createApplicationFacts struct{ gatewayport.ApplicationStore }

func (createApplicationFacts) ApplicationByProjectAndName(context.Context, string, string) (model.Application, error) {
	return model.Application{}, repository.ErrNotFound
}
func (createApplicationFacts) ApplicationByProjectAndCode(context.Context, string, string) (model.Application, error) {
	return model.Application{}, repository.ErrNotFound
}

type createTransaction struct{}

func (createTransaction) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestCreateGatewayRequiresFreshEnvironmentProbe(t *testing.T) {
	image, entrypoint, tlsMode := "traefik:3.6", "web", "none"
	timeout := 20
	service := New(createProjectFacts{}, createEnvironmentFacts{environment: model.Environment{Id: "env-1", ProjectId: "project-1", TargetRevision: 2}}, createApplicationFacts{}, nil, nil, nil, createTransaction{})
	_, err := service.CreateGateway(context.Background(), "user-1", "project-1", gatewaydto.GatewayCreateInput{
		Code: "traefik", Name: "Traefik", InitialComponentImage: &image,
		RestApiUrl: model.GatewayRestApiContainerUrl, RestApiHostUrl: model.GatewayRestApiHostUrl,
		InternalDomain: "example.test", DefaultEntrypoint: &entrypoint, TLSMode: &tlsMode,
		RestReadyTimeoutSeconds: &timeout,
	})
	if err == nil || !apperror.IsKind(err, apperror.KindValidation) || !strings.Contains(err.Error(), "must pass probe") {
		t.Fatalf("CreateGateway error = %v, want fresh Probe validation", err)
	}
}
