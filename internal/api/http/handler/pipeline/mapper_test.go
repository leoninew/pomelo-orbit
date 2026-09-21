package pipelinehandler

import (
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/application/variableview"
	commonv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/common"
)

func TestVariableResponsesIncludeStageBindingAndConfiguration(t *testing.T) {
	responses := variableResponses([]variableview.View{{
		Name: "working_dir", Kind: variableview.KindPipelineContext, Scope: "stage", StageBinding: &variableview.StageBinding{StageId: "stage-frontend", StageName: "frontend"},
		GlobalConfiguration: &variableview.Configuration{Value: "shared"},
		StageOverride:       &variableview.Configuration{Value: "frontend"},
	}})
	if len(responses) != 1 || responses[0].GetStageBinding().GetStageId() != "stage-frontend" {
		t.Fatalf("responses=%#v", responses)
	}
	if responses[0].GetKind() != string(variableview.KindPipelineContext) {
		t.Fatalf("kind=%q", responses[0].GetKind())
	}
	if responses[0].GetGlobalConfiguration().GetValue().GetStringValue() != "shared" || responses[0].GetStageOverride().GetValue().GetStringValue() != "frontend" {
		t.Fatalf("response=%#v", responses[0])
	}
}

func TestVariableRequestMapsKeepStageScope(t *testing.T) {
	items := variableRequestMaps([]*commonv1.VariableDeclarationReq{{Name: "working_dir", StageId: "stage-build"}})
	if len(items) != 1 || items[0]["name"] != "working_dir" || items[0]["stage_id"] != "stage-build" {
		t.Fatalf("items=%#v", items)
	}
}
