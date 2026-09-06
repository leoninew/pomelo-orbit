package pipelinerunsvc

import (
	"strings"
	"testing"

	pipelinevariable "github.com/leoninew/pomelo-orbit/internal/application/pipeline/rule/pipelinevariable"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestResolveStagesUsesEachStageLiquidDefault(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1"}
	stages := []model.StageDefinition{
		{Id: "stage-frontend", Name: "frontend", Script: `cd {{ working_dir | default: "frontend" }}`},
		{Id: "stage-backend", Name: "backend", Script: `cd {{ working_dir | default: "backend" }}`},
	}

	_, variables, err := pipelinevariable.ResolveRuntimeVariables(repo, pipeline, stages, pipelinevariable.RuntimeVariableOverrides{})
	if err != nil {
		t.Fatalf("resolve runtime variables: %v", err)
	}
	resolved, err := resolveStages(stages, variables)
	if err != nil {
		t.Fatalf("resolve stages: %v", err)
	}
	if resolved[0].Script != "cd frontend" || resolved[1].Script != "cd backend" {
		t.Fatalf("resolved scripts=%q, %q", resolved[0].Script, resolved[1].Script)
	}
}

func TestResolveStagesUsesPipelineValueBeforeStageDefault(t *testing.T) {
	stages := []model.StageDefinition{{Id: "stage-build", Name: "build", Script: `cd {{ working_dir | default: "frontend" }}`}}
	resolved, err := resolveStages(stages, pipelinevariable.RuntimeVariables{Global: map[string]any{"working_dir": "shared"}})
	if err != nil {
		t.Fatalf("resolve stages: %v", err)
	}
	if resolved[0].Script != "cd shared" {
		t.Fatalf("resolved script=%q", resolved[0].Script)
	}
}

func TestResolveStagesReportsStageAndArtifactField(t *testing.T) {
	stages := []model.StageDefinition{{
		Id: "stage-build", Name: "build", Script: "echo build",
		Artifacts: []model.ArtifactConfig{{Name: "image", Collector: "docker_image", Reference: `registry/{{ IMAGE_TAG }}`}},
	}}
	_, err := resolveStages(stages, pipelinevariable.RuntimeVariables{Global: map[string]any{}})
	if err == nil || !strings.Contains(err.Error(), "stage build artifacts[0].reference") || !strings.Contains(err.Error(), "IMAGE_TAG") {
		t.Fatalf("expected stage artifact context, got %v", err)
	}
}

func TestResolveStagesUsesStageScopedPipelineValues(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1", VariableDeclarations: `[{"name":"working_dir","stage_id":"stage-frontend","value":"web-dist"}]`}
	stages := []model.StageDefinition{
		{Id: "stage-frontend", Name: "frontend", Script: `cd {{ working_dir | default: "web" }}`},
		{Id: "stage-backend", Name: "backend", Script: `cd {{ working_dir | default: "webapi" }}`},
	}

	declarations, runtime, err := pipelinevariable.ResolveRuntimeVariables(repo, pipeline, stages, pipelinevariable.RuntimeVariableOverrides{})
	if err != nil {
		t.Fatalf("resolve runtime variables: %v", err)
	}
	if runtime.ValuesForStage(stages[0])["working_dir"] != "web-dist" {
		t.Fatalf("frontend values=%#v", runtime.ValuesForStage(stages[0]))
	}
	if _, ok := runtime.ValuesForStage(stages[1])["working_dir"]; ok {
		t.Fatalf("backend must retain its Liquid default: %#v", runtime.ValuesForStage(stages[1]))
	}
	resolved, err := resolveStages(stages, runtime)
	if err != nil {
		t.Fatalf("resolve stages: %v", err)
	}
	if resolved[0].Script != "cd web-dist" || resolved[1].Script != "cd webapi" {
		t.Fatalf("resolved scripts=%q, %q", resolved[0].Script, resolved[1].Script)
	}
	data, err := pipelinevariable.MarshalRuntimeVariableSnapshot(runtime, declarations)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	_, restored, err := pipelinevariable.UnmarshalRuntimeVariableSnapshot(data)
	if err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	if restored.ValuesForStage(stages[0])["working_dir"] != "web-dist" {
		t.Fatalf("restored frontend values=%#v", restored.ValuesForStage(stages[0]))
	}
}
