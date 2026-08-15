package pipelinevariable

import (
	"strings"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestResolvePipelineVariableDeclarationsKeepsPipelineConfigurationAndStageDeclarations(t *testing.T) {
	stages := []model.PipelineStage{{Name: "build", Script: "cd {{ working_dir }}\ndocker build -f {{ repository_dockerfile }} ."}}
	declarations, err := ResolvePipelineVariableDeclarations(stages, `[{"name":"working_dir","default":".","source":"pipeline_custom","editable":true}]`)
	if err != nil {
		t.Fatalf("ResolvePipelineVariableDeclarations returned error: %v", err)
	}
	byName := make(map[string]model.VariableDeclaration, len(declarations))
	for _, declaration := range declarations {
		byName[declaration.Name] = declaration
	}
	if got := byName["working_dir"].Default; got != "." || byName["working_dir"].Source != "pipeline_custom" {
		t.Fatalf("working_dir = %#v", byName["working_dir"])
	}
	if declaration, ok := byName["repository_dockerfile"]; !ok || declaration.Source != "pipeline_stage" || declaration.Editable {
		t.Fatalf("repository_dockerfile = %#v", declaration)
	}
	if declaration, ok := byName["repository_ref"]; !ok || declaration.Source != "pipeline" || declaration.Editable {
		t.Fatalf("repository_ref = %#v", declaration)
	}
}

func TestResolvePipelineVariableDeclarationsUsesStageDefaultAfterPipelineConfigurationIsDeleted(t *testing.T) {
	stages := []model.PipelineStage{{Name: "build", Script: `echo {{ IMAGE_TAG | default: "stage" }}`}}
	declarations, err := ResolvePipelineVariableDeclarations(stages, `[]`)
	if err != nil {
		t.Fatalf("ResolvePipelineVariableDeclarations returned error: %v", err)
	}
	byName := make(map[string]model.VariableDeclaration, len(declarations))
	for _, declaration := range declarations {
		byName[declaration.Name] = declaration
	}
	if declaration := byName["IMAGE_TAG"]; declaration.Default != "stage" || declaration.Source != "pipeline_stage" {
		t.Fatalf("declarations = %#v", declarations)
	}
	if declaration, ok := byName["repository_ref"]; !ok || declaration.Editable {
		t.Fatalf("repository_ref = %#v", declaration)
	}
}

func TestResolveRuntimeVariablesKeepsAllDeclarationsAndAppliesPersistedPrecedence(t *testing.T) {
	repo := model.Repository{
		Id: "repo-1", Name: "Repo", Code: "repo", RepositoryUrl: "https://example.invalid/repo.git", DefaultBranch: "main",
		VariableOverrides: `[{"name":"IMAGE_TAG","value":"repo"}]`,
	}
	pipeline := model.Pipeline{Id: "pipeline-1", Name: "Pipeline", Version: 3, VariableDeclarations: `[{"name":"IMAGE_TAG","default":"pipeline","value":null},{"name":"UNUSED","default":"kept"}]`}
	stages := []model.StageDefinition{{Name: "build", Script: `echo {{ IMAGE_TAG | default: "stage" }}`}}
	declarations, values, err := ResolveRuntimeVariables(repo, pipeline, stages, nil)
	if err != nil {
		t.Fatalf("resolve runtime variables: %v", err)
	}
	if values["IMAGE_TAG"] != "repo" || values["UNUSED"] != "kept" || values["repository_ref"] != "main" {
		t.Fatalf("resolved values=%#v", values)
	}
	byName := make(map[string]model.VariableDeclaration, len(declarations))
	for _, declaration := range declarations {
		if _, exists := byName[declaration.Name]; exists {
			t.Fatalf("duplicate declaration %q", declaration.Name)
		}
		byName[declaration.Name] = declaration
	}
	if byName["IMAGE_TAG"].Source != "repository" || byName["UNUSED"].Source != "pipeline" || byName["repository_ref"].Source != "runtime" || byName["repository_ref"].Editable || byName["runtime_datetime"].Editable {
		t.Fatalf("declarations=%#v", byName)
	}
}

func TestResolveRuntimeVariablesUsesLowerPriorityDeclarationWhenRepositoryHasNoValue(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main", VariableOverrides: `[{"name":"IMAGE_TAG"}]`}
	pipeline := model.Pipeline{Id: "pipeline-1", VariableDeclarations: `[{"name":"IMAGE_TAG","value":"pipeline"}]`}
	declarations, values, err := ResolveRuntimeVariables(repo, pipeline, nil, nil)
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

func TestResolveRuntimeVariablesRejectsUnknownAndSystemRetryOverrides(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1", Name: "Pipeline", Version: 1}
	for _, overrides := range []map[string]string{{"runtime_datetime": "bad"}, {"UNKNOWN": "bad"}} {
		if _, _, err := ResolveRuntimeVariables(repo, pipeline, nil, overrides); err == nil {
			t.Fatalf("overrides %#v should be rejected", overrides)
		}
	}
}

func TestResolveRuntimeVariablesAllowsInternalRetryToPreserveRepositoryRef(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1"}
	_, values, err := ResolveRuntimeVariables(repo, pipeline, nil, map[string]string{"repository_ref": "release"})
	if err != nil {
		t.Fatalf("resolve runtime variables: %v", err)
	}
	if values["repository_ref"] != "release" {
		t.Fatalf("repository_ref=%#v, want release", values["repository_ref"])
	}
}

func TestResolveRuntimeVariablesUsesStageDefaultAndIncludesSystemValues(t *testing.T) {
	repo := model.Repository{Id: "repo-1", Name: "Repo", Code: "repo", RepositoryUrl: "https://example.invalid/repo.git", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1", Name: "Pipeline", Version: 2, VariableDeclarations: `[{"name":"UNUSED","default":"pipeline-default"}]`}
	stages := []model.StageDefinition{{Name: "build", Script: `echo {{ IMAGE_TAG | default: "latest" }}`}}

	declarations, values, err := ResolveRuntimeVariables(repo, pipeline, stages, nil)
	if err != nil {
		t.Fatalf("resolve runtime variables: %v", err)
	}
	if values["IMAGE_TAG"] != "latest" || values["UNUSED"] != "pipeline-default" || values["repository_ref"] != "main" {
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
	if len(snapshot) != len(declarations) || persisted["repository_ref"] != "main" || persisted["repository_code"] != repo.Code {
		t.Fatalf("persisted runtime snapshot=%#v, values=%#v", snapshot, persisted)
	}
}

func TestResolveRuntimeVariablesRequiresFinalValue(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1", VariableDeclarations: `[{"name":"IMAGE_TAG"}]`}
	_, _, err := ResolveRuntimeVariables(repo, pipeline, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "Missing variable value: IMAGE_TAG") {
		t.Fatalf("expected missing resolved variable error, got %v", err)
	}
}

func TestResolveRuntimeVariablesFromPipelineStagesUsesPersistedConfiguration(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1", VariableDeclarations: `[{"name":"IMAGE_TAG","value":"pipeline"}]`}
	stages := []model.PipelineStage{{Name: "build", Script: `echo {{ IMAGE_TAG | default: "stage" }}`}}
	_, values, err := ResolveRuntimeVariablesFromPipelineStages(repo, pipeline, stages, nil)
	if err != nil {
		t.Fatalf("resolve runtime variables from pipeline stages: %v", err)
	}
	if values["IMAGE_TAG"] != "pipeline" || values["repository_ref"] != "main" {
		t.Fatalf("resolved values=%#v", values)
	}
}

func TestNormalizePipelineVariablesRejectsInvalidNamesAndDuplicates(t *testing.T) {
	for _, variables := range [][]map[string]any{
		{{"name": "repository_ref"}},
		{{"name": "bad-name"}},
		{{"name": "IMAGE_TAG"}, {"name": "IMAGE_TAG"}},
	} {
		if _, err := NormalizePipelineVariables(variables); err == nil {
			t.Fatalf("variables %#v should be rejected", variables)
		}
	}
}

func TestResolveTemplatePipelineVariablesPreservesTemplateDetailContract(t *testing.T) {
	stages := []model.PipelineStage{{Name: "build", Script: "echo {{ IMAGE_TAG }}"}}
	variables := []map[string]any{{"name": "invalid-name", "value": "allowed-for-template"}}

	resolved := ResolveTemplatePipelineVariables(stages, variables)
	byName := make(map[string]map[string]any, len(resolved))
	for _, variable := range resolved {
		name, _ := variable["name"].(string)
		byName[name] = variable
	}
	if byName["IMAGE_TAG"]["source"] != "pipeline_stage" || byName["IMAGE_TAG"]["editable"] != true {
		t.Fatalf("stage declaration=%#v", byName["IMAGE_TAG"])
	}
	if _, exists := byName["repository_ref"]; !exists {
		t.Fatalf("template declarations=%#v", byName)
	}
	if byName["invalid-name"]["value"] != "allowed-for-template" {
		t.Fatalf("template custom declaration=%#v", byName["invalid-name"])
	}
}
