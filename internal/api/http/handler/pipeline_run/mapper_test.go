package pipelinerunhandler

import (
	"testing"

	pipelinerundto "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestPipelineRunResponseShowsTargetWithoutCredentials(t *testing.T) {
	environmentId := "environment-1"
	targetType := model.EnvironmentTargetTypeSSH
	credentialId := "credential-1"
	response := pipelineRunResponse(pipelinerundto.PipelineRunDetail{Run: model.PipelineRun{
		Id: "run-1", EnvironmentId: &environmentId, EnvironmentTargetType: &targetType, SSHCredentialId: &credentialId,
	}})
	if response.EnvironmentId == nil || *response.EnvironmentId != environmentId ||
		response.EnvironmentTargetType == nil || *response.EnvironmentTargetType != targetType {
		t.Fatalf("pipeline run target response = %+v", response)
	}
	if response.GetEnvironmentTargetType() != model.EnvironmentTargetTypeSSH {
		t.Fatalf("target type = %q", response.GetEnvironmentTargetType())
	}
	legacy := pipelineRunResponse(pipelinerundto.PipelineRunDetail{Run: model.PipelineRun{Id: "legacy"}})
	if legacy.EnvironmentId != nil || legacy.EnvironmentTargetType != nil {
		t.Fatalf("historical run unexpectedly gained target = %+v", legacy)
	}
}
