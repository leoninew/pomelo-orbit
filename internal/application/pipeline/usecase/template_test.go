package pipelinesvc

import (
	"testing"

	pipelinevariable "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/rule/pipelinevariable"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestResolveStagesRendersScriptAndArtifacts(t *testing.T) {
	stages := []model.StageDefinition{{
		Id:     "stage-1",
		Name:   "build",
		Script: "{% if PUSH_IMAGE %}docker build -t {{ IMAGE }} .{% endif %}",
		Artifacts: []model.ArtifactConfig{
			{Collector: "docker_image", Name: "{{ IMAGE }}", Reference: "{{ IMAGE }}:latest"},
		},
	}}

	resolved, err := resolveStages(stages, map[string]any{"IMAGE": "demo", "PUSH_IMAGE": true})
	if err != nil {
		t.Fatal(err)
	}
	if resolved[0].Script != "docker build -t demo ." {
		t.Fatalf("unexpected script: %s", resolved[0].Script)
	}
	if resolved[0].Artifacts[0].Name != "demo" || resolved[0].Artifacts[0].Reference != "demo:latest" {
		t.Fatalf("unexpected artifact: %+v", resolved[0].Artifacts[0])
	}
}

func TestResolveStagesRejectsMissingVariable(t *testing.T) {
	_, err := resolveStages([]model.StageDefinition{{Id: "stage-1", Script: "{{ MISSING }}"}}, map[string]any{})
	if err == nil {
		t.Fatal("expected missing variable error")
	}
}

func TestResolveTemplateVariablesExtractsLiquidDefault(t *testing.T) {
	variables := pipelinevariable.ResolveTemplateVariables([]model.PipelineStage{{Script: "cd {{ working_dir | default: '.' }}"}}, nil)
	if len(variables) != 1 {
		t.Fatalf("expected one variable, got %+v", variables)
	}
	if variables[0]["name"] != "working_dir" || variables[0]["default"] != "." {
		t.Fatalf("unexpected variable: %+v", variables[0])
	}
}
