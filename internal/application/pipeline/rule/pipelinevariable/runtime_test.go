package pipelinevariable

import (
	"strings"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestResolvePipelineVariableDeclarationsUsesPipelineAndStageDeclarations(t *testing.T) {
	stages := []model.PipelineStage{{Name: "build", Script: "cd {{ working_dir }}\ndocker build -f {{ repository_dockerfile }} ."}}
	declarations, err := ResolvePipelineVariableDeclarations(stages, `[{"name":"working_dir","default":".","source":"pipeline_custom","editable":true}]`)
	if err != nil {
		t.Fatalf("ResolvePipelineVariableDeclarations returned error: %v", err)
	}
	byName := make(map[string]model.VariableDeclaration, len(declarations))
	for _, declaration := range declarations {
		byName[declaration.Name] = declaration
	}
	if got := byName["working_dir"].Default; got != "." {
		t.Fatalf("working_dir default = %#v, want .", got)
	}
	if _, ok := byName["repository_dockerfile"]; !ok {
		t.Fatal("repository_dockerfile was not extracted from the stage")
	}
}

func TestResolveRuntimeVariablesKeepsAllDeclarationsAndAppliesPrecedence(t *testing.T) {
	repo := model.Repository{
		Id: "repo-1", Name: "Repo", Code: "repo", RepositoryUrl: "https://example.invalid/repo.git", DefaultBranch: "main",
		VariableOverrides: `[{"name":"IMAGE_TAG","value":"repo"}]`,
	}
	pipeline := model.Pipeline{Id: "pipeline-1", Name: "Pipeline", Version: 3, VariableDeclarations: `[{"name":"IMAGE_TAG","default":"pipeline","value":null},{"name":"UNUSED","default":"kept"}]`}
	stages := []model.StageDefinition{{Name: "build", Script: `echo {{ IMAGE_TAG | default: "stage" }}`}}
	declarations, values, err := ResolveRuntimeVariables(repo, pipeline, stages, map[string]string{"repository_ref": "release", "IMAGE_TAG": "form", "UNUSED": "kept"}, true)
	if err != nil {
		t.Fatalf("resolve runtime variables: %v", err)
	}
	if values["IMAGE_TAG"] != "form" || values["UNUSED"] != "kept" || values["repository_ref"] != "release" {
		t.Fatalf("resolved values=%#v", values)
	}
	byName := make(map[string]model.VariableDeclaration, len(declarations))
	for _, declaration := range declarations {
		if _, exists := byName[declaration.Name]; exists {
			t.Fatalf("duplicate declaration %q", declaration.Name)
		}
		byName[declaration.Name] = declaration
	}
	if byName["IMAGE_TAG"].Source != "repository" || byName["UNUSED"].Source != "pipeline" || byName["repository_ref"].Source != "runtime" || byName["runtime_datetime"].Editable {
		t.Fatalf("declarations=%#v", byName)
	}
}

func TestResolveRuntimeVariablesUsesLowerPriorityDeclarationWhenHigherHasNoValue(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main", VariableOverrides: `[{"name":"IMAGE_TAG"}]`}
	pipeline := model.Pipeline{Id: "pipeline-1", VariableDeclarations: `[{"name":"IMAGE_TAG","value":"pipeline"}]`}
	declarations, values, err := ResolveRuntimeVariables(repo, pipeline, nil, map[string]string{}, false)
	if err != nil {
		t.Fatalf("resolve runtime variables: %v", err)
	}
	if values["IMAGE_TAG"] != "pipeline" {
		t.Fatalf("IMAGE_TAG=%#v, want pipeline", values["IMAGE_TAG"])
	}
	for _, declaration := range declarations {
		if declaration.Name == "IMAGE_TAG" && declaration.Source != "repository" {
			t.Fatalf("IMAGE_TAG source=%q, want repository", declaration.Source)
		}
	}
}

func TestResolveRuntimeVariablesRejectsSystemAndUnknownFormKeys(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1", Name: "Pipeline", Version: 1}
	stages := []model.StageDefinition{}
	for _, form := range []map[string]string{{"runtime_datetime": "bad"}, {"UNKNOWN": "bad"}} {
		if _, _, err := ResolveRuntimeVariables(repo, pipeline, stages, form, false); err == nil {
			t.Fatalf("form %#v should be rejected", form)
		}
	}
}

func TestResolveRuntimeVariablesUsesStageDefaultAndIncludesRuntimeAndSystemValues(t *testing.T) {
	repo := model.Repository{Id: "repo-1", Name: "Repo", Code: "repo", RepositoryUrl: "https://example.invalid/repo.git", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1", Name: "Pipeline", Version: 2, VariableDeclarations: `[{"name":"UNUSED","default":"pipeline-default"}]`}
	stages := []model.StageDefinition{{Name: "build", Script: `echo {{ IMAGE_TAG | default: "latest" }}`}}

	declarations, values, err := ResolveRuntimeVariables(repo, pipeline, stages, map[string]string{"repository_ref": "release"}, false)
	if err != nil {
		t.Fatalf("resolve runtime variables: %v", err)
	}
	if values["IMAGE_TAG"] != "latest" || values["UNUSED"] != "pipeline-default" || values["repository_ref"] != "release" {
		t.Fatalf("resolved values=%#v", values)
	}
	if values["repository_code"] != repo.Code || !HasRuntimeValue(values["runtime_datetime"]) {
		t.Fatalf("system values=%#v", values)
	}
	data, err := MarshalRuntimeVariableSnapshot(values, declarations)
	if err != nil {
		t.Fatalf("marshal runtime snapshot: %v", err)
	}
	snapshot, persisted, err := UnmarshalRuntimeVariableSnapshot(data)
	if err != nil {
		t.Fatalf("unmarshal runtime snapshot: %v", err)
	}
	if len(snapshot) != len(declarations) || persisted["repository_ref"] != "release" || persisted["repository_code"] != repo.Code {
		t.Fatalf("persisted runtime snapshot=%#v, values=%#v", snapshot, persisted)
	}
}

func TestResolveRuntimeVariablesRequiresEveryConfigurableValueForTrigger(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1", VariableDeclarations: `[{"name":"IMAGE_TAG"}]`}
	_, _, err := ResolveRuntimeVariables(repo, pipeline, nil, map[string]string{"repository_ref": "main"}, true)
	if err == nil || !strings.Contains(err.Error(), "Missing variable value: IMAGE_TAG") {
		t.Fatalf("expected missing configurable variable error, got %v", err)
	}
}

func TestResolveRuntimeVariablesRejectsAnEmptySubmittedValueForTrigger(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1", VariableDeclarations: `[{"name":"IMAGE_TAG","default":"latest"}]`}
	_, _, err := ResolveRuntimeVariables(repo, pipeline, nil, map[string]string{"repository_ref": "main", "IMAGE_TAG": ""}, true)
	if err == nil || !strings.Contains(err.Error(), "Missing variable value: IMAGE_TAG") {
		t.Fatalf("expected empty submitted value error, got %v", err)
	}
}
