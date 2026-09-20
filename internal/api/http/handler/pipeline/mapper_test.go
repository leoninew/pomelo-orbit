package pipelinehandler

import (
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/application/variableview"
	commonv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/common"
)

func TestVariableResponsesIncludeStageBindingAndReferences(t *testing.T) {
	responses := variableResponses([]variableview.View{{
		Name: "working_dir", Kind: "stage_variable", Scope: "stage", StageBinding: &variableview.StageBinding{StageId: "stage-frontend", StageName: "frontend"},
		References:          []variableview.Reference{{StageId: "stage-frontend", StageName: "frontend", Field: "artifact_reference", ArtifactName: "web", Default: "frontend", HasDefault: true}},
		GlobalConfiguration: &variableview.Configuration{Value: "shared"},
		StageOverride:       &variableview.Configuration{Value: "frontend"},
	}})
	if len(responses) != 1 || responses[0].GetStageBinding().GetStageId() != "stage-frontend" || len(responses[0].References) != 1 {
		t.Fatalf("responses=%#v", responses)
	}
	if responses[0].References[0].GetField() != "artifact_reference" || responses[0].References[0].GetDefault().GetStringValue() != "frontend" || responses[0].GetGlobalConfiguration().GetValue().GetStringValue() != "shared" || responses[0].GetStageOverride().GetValue().GetStringValue() != "frontend" {
		t.Fatalf("response=%#v", responses[0])
	}
}

func TestVariableRequestMapsKeepStageScope(t *testing.T) {
	items := variableRequestMaps([]*commonv1.VariableDeclarationReq{{Name: "working_dir", StageId: "stage-build"}})
	if len(items) != 1 || items[0]["name"] != "working_dir" || items[0]["stage_id"] != "stage-build" {
		t.Fatalf("items=%#v", items)
	}
}
