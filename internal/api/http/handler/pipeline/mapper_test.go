package pipelinehandler

import (
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestVariableDeclarationResponsesIncludeStageDefaults(t *testing.T) {
	responses := variableDeclarationResponses([]model.VariableDeclaration{{
		Name: "working_dir", Source: "pipeline_stage", StageId: "stage-frontend", StageName: "frontend",
		StageDefaults: []model.StageVariableDefault{
			{StageId: "stage-frontend", StageName: "frontend", Default: "frontend"},
			{StageId: "stage-backend", StageName: "backend", Default: "backend"},
		},
	}})
	if len(responses) != 1 || len(responses[0].StageDefaults) != 2 || responses[0].StageId != "stage-frontend" || responses[0].StageName != "frontend" {
		t.Fatalf("responses=%#v", responses)
	}
	if responses[0].StageDefaults[0].StageId != "stage-frontend" || responses[0].StageDefaults[0].Default.GetStringValue() != "frontend" || responses[0].StageDefaults[1].StageName != "backend" || responses[0].StageDefaults[1].Default.GetStringValue() != "backend" {
		t.Fatalf("stage defaults=%#v", responses[0].StageDefaults)
	}
}

func TestVariableDeclarationsMapStageDefaults(t *testing.T) {
	declarations, err := variableDeclarations([]map[string]any{{
		"name": "working_dir", "source": "pipeline_custom", "stage_id": "stage-build", "stage_name": "build", "stage_defaults": []model.StageVariableDefault{{StageId: "stage-build", StageName: "build", Default: "backend"}},
	}})
	if err != nil {
		t.Fatalf("map variable declarations: %v", err)
	}
	if len(declarations) != 1 || len(declarations[0].StageDefaults) != 1 || declarations[0].StageId != "stage-build" || declarations[0].StageName != "build" {
		t.Fatalf("declarations=%#v", declarations)
	}
}
