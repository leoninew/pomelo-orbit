package pipelinevariable

import (
	"reflect"
	"strings"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestExtractStageVariableReferences(t *testing.T) {
	references, err := extractStageVariableReferences("cd {{ working_dir | default: \"frontend|admin\" }}\necho {{ IMAGE_TAG }}\necho {{ name | upcase }}")
	if err != nil {
		t.Fatalf("extract references: %v", err)
	}
	want := []stageVariableReference{{Name: "working_dir", HasDefault: true, Default: "frontend|admin"}, {Name: "IMAGE_TAG"}}
	if !reflect.DeepEqual(references, want) {
		t.Fatalf("references = %#v, want %#v", references, want)
	}
}

func TestValidateStageVariableExpressionsRejectsUnsupportedDefaults(t *testing.T) {
	for _, input := range []string{
		"echo ${working_dir:-frontend}",
		"echo {{ working_dir | default: frontend }}",
	} {
		if err := ValidateStageVariableExpressions(input); err == nil {
			t.Fatalf("expected unsupported expression error for %q", input)
		}
	}
}

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
	if declaration, ok := byName["repository_dockerfile"]; !ok || declaration.Source != "pipeline_stage" || declaration.Editable || declaration.StageName != "build" {
		t.Fatalf("repository_dockerfile = %#v", declaration)
	}
	if declaration, ok := byName["repository_ref"]; !ok || declaration.Source != "pipeline" || declaration.Editable {
		t.Fatalf("repository_ref = %#v", declaration)
	}
}

func TestResolvePipelineVariableDeclarationsUsesStageDefaultAfterPipelineConfigurationIsDeleted(t *testing.T) {
	stages := []model.PipelineStage{{Id: "stage-build", Name: "build", Script: `echo {{ IMAGE_TAG | default: "stage" }}`}}
	declarations, err := ResolvePipelineVariableDeclarations(stages, `[]`)
	if err != nil {
		t.Fatalf("ResolvePipelineVariableDeclarations returned error: %v", err)
	}
	byName := make(map[string]model.VariableDeclaration, len(declarations))
	for _, declaration := range declarations {
		byName[declaration.Name] = declaration
	}
	if declaration := byName["IMAGE_TAG"]; declaration.Default != nil || declaration.Source != "pipeline_stage" || len(declaration.StageDefaults) != 1 || declaration.StageDefaults[0].StageId != "stage-build" || declaration.StageDefaults[0].StageName != "build" || declaration.StageDefaults[0].Default != "stage" {
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
	declarations, values, err := ResolveRuntimeVariables(repo, pipeline, stages, RuntimeVariableOverrides{})
	if err != nil {
		t.Fatalf("resolve runtime variables: %v", err)
	}
	if values.Global["IMAGE_TAG"] != "repo" || values.Global["UNUSED"] != "kept" || values.Global["repository_ref"] != "main" {
		t.Fatalf("resolved values=%#v", values)
	}
	byScope := make(map[string]model.VariableDeclaration, len(declarations))
	for _, declaration := range declarations {
		key := declaration.Name + "\x00" + declaration.StageId
		if _, exists := byScope[key]; exists {
			t.Fatalf("duplicate declaration %q", key)
		}
		byScope[key] = declaration
	}
	if byScope["IMAGE_TAG\x00"].Source != "repository" || byScope["UNUSED\x00"].Source != "pipeline" || byScope["repository_ref\x00"].Source != "runtime" || byScope["repository_ref\x00"].Editable || byScope["runtime_datetime\x00"].Editable {
		t.Fatalf("declarations=%#v", byScope)
	}
}

func TestResolveRuntimeVariablesUsesLowerPriorityDeclarationWhenRepositoryHasNoValue(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main", VariableOverrides: `[{"name":"IMAGE_TAG"}]`}
	pipeline := model.Pipeline{Id: "pipeline-1", VariableDeclarations: `[{"name":"IMAGE_TAG","value":"pipeline"}]`}
	declarations, values, err := ResolveRuntimeVariables(repo, pipeline, nil, RuntimeVariableOverrides{})
	if err != nil {
		t.Fatalf("resolve runtime variables: %v", err)
	}
	if values.Global["IMAGE_TAG"] != "pipeline" {
		t.Fatalf("IMAGE_TAG=%#v, want pipeline", values.Global["IMAGE_TAG"])
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
	for _, overrides := range []RuntimeVariableOverrides{{Global: map[string]string{"runtime_datetime": "bad"}}, {Global: map[string]string{"UNKNOWN": "bad"}}} {
		if _, _, err := ResolveRuntimeVariables(repo, pipeline, nil, overrides); err == nil {
			t.Fatalf("overrides %#v should be rejected", overrides)
		}
	}
}

func TestResolveRuntimeVariablesAllowsInternalRetryToPreserveRepositoryRef(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1"}
	_, values, err := ResolveRuntimeVariables(repo, pipeline, nil, RuntimeVariableOverrides{Global: map[string]string{"repository_ref": "release"}})
	if err != nil {
		t.Fatalf("resolve runtime variables: %v", err)
	}
	if values.Global["repository_ref"] != "release" {
		t.Fatalf("repository_ref=%#v, want release", values.Global["repository_ref"])
	}
}

func TestResolveRuntimeVariablesKeepsStageDefaultOutOfGlobalValues(t *testing.T) {
	repo := model.Repository{Id: "repo-1", Name: "Repo", Code: "repo", RepositoryUrl: "https://example.invalid/repo.git", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1", Name: "Pipeline", Version: 2, VariableDeclarations: `[{"name":"UNUSED","default":"pipeline-default"}]`}
	stages := []model.StageDefinition{{Id: "stage-build", Name: "build", Script: `echo {{ IMAGE_TAG | default: "latest" }}`}}

	declarations, values, err := ResolveRuntimeVariables(repo, pipeline, stages, RuntimeVariableOverrides{})
	if err != nil {
		t.Fatalf("resolve runtime variables: %v", err)
	}
	if _, exists := values.Global["IMAGE_TAG"]; exists {
		t.Fatalf("stage default must not be promoted to global values: %#v", values)
	}
	if values.Global["UNUSED"] != "pipeline-default" || values.Global["repository_ref"] != "main" {
		t.Fatalf("resolved values=%#v", values)
	}
	byName := make(map[string]model.VariableDeclaration, len(declarations))
	for _, declaration := range declarations {
		byName[declaration.Name] = declaration
	}
	if defaults := byName["IMAGE_TAG"].StageDefaults; len(defaults) != 1 || defaults[0].StageName != "build" || defaults[0].Default != "latest" {
		t.Fatalf("stage defaults=%#v", defaults)
	}
	if values.Global["repository_code"] != repo.Code || !HasRuntimeValue(values.Global["runtime_datetime"]) {
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
	if _, exists := persisted.Global["IMAGE_TAG"]; exists {
		t.Fatalf("stage default must not be persisted as a global value: %#v", persisted)
	}
	if len(snapshot) != len(declarations) || persisted.Global["repository_ref"] != "main" || persisted.Global["repository_code"] != repo.Code {
		t.Fatalf("persisted runtime snapshot=%#v, values=%#v", snapshot, persisted)
	}
}

func TestResolveRuntimeVariablesRequiresFinalValue(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1", VariableDeclarations: `[{"name":"IMAGE_TAG"}]`}
	_, _, err := ResolveRuntimeVariables(repo, pipeline, nil, RuntimeVariableOverrides{})
	if err == nil || !strings.Contains(err.Error(), "Missing variable value: IMAGE_TAG") {
		t.Fatalf("expected missing resolved variable error, got %v", err)
	}
}

func TestResolveRuntimeVariablesFromPipelineStagesUsesPersistedConfiguration(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1", VariableDeclarations: `[{"name":"IMAGE_TAG","value":"pipeline"}]`}
	stages := []model.PipelineStage{{Name: "build", Script: `echo {{ IMAGE_TAG | default: "stage" }}`}}
	_, values, err := ResolveRuntimeVariablesFromPipelineStages(repo, pipeline, stages, RuntimeVariableOverrides{})
	if err != nil {
		t.Fatalf("resolve runtime variables from pipeline stages: %v", err)
	}
	if values.Global["IMAGE_TAG"] != "pipeline" || values.Global["repository_ref"] != "main" {
		t.Fatalf("resolved values=%#v", values)
	}
}

func TestResolveRuntimeVariablesPreservesMultipleStageDefaults(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1"}
	stages := []model.StageDefinition{
		{Id: "stage-frontend", Name: "frontend", Script: `cd {{ working_dir | default: "frontend" }}`},
		{Id: "stage-backend", Name: "backend", Script: `cd {{ working_dir | default: "backend" }}`},
	}

	declarations, values, err := ResolveRuntimeVariables(repo, pipeline, stages, RuntimeVariableOverrides{})
	if err != nil {
		t.Fatalf("resolve runtime variables: %v", err)
	}
	if _, exists := values.Global["working_dir"]; exists {
		t.Fatalf("stage defaults must not be promoted to global values: %#v", values)
	}
	byScope := make(map[string]model.VariableDeclaration, len(declarations))
	for _, declaration := range declarations {
		byScope[declaration.Name+"\x00"+declaration.StageId] = declaration
	}
	frontend, backend := byScope["working_dir\x00stage-frontend"], byScope["working_dir\x00stage-backend"]
	if len(frontend.StageDefaults) != 1 || frontend.StageDefaults[0].Default != "frontend" || len(backend.StageDefaults) != 1 || backend.StageDefaults[0].Default != "backend" {
		t.Fatalf("working_dir declarations=%#v", byScope)
	}
}

func TestResolveRuntimeVariablesPipelineValueOverridesEveryStageDefault(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1", VariableDeclarations: `[{"name":"working_dir","value":"shared"}]`}
	stages := []model.StageDefinition{
		{Id: "stage-frontend", Name: "frontend", Script: `cd {{ working_dir | default: "frontend" }}`},
		{Id: "stage-backend", Name: "backend", Script: `cd {{ working_dir | default: "backend" }}`},
	}

	_, values, err := ResolveRuntimeVariables(repo, pipeline, stages, RuntimeVariableOverrides{})
	if err != nil {
		t.Fatalf("resolve runtime variables: %v", err)
	}
	if values.Global["working_dir"] != "shared" {
		t.Fatalf("working_dir=%#v, want shared", values.Global["working_dir"])
	}
}

func TestResolveRuntimeVariablesRetryOverridesTakePrecedenceOverRepositoryVariables(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main", VariableOverrides: `[{"name":"working_dir","value":"repository"}]`}
	pipeline := model.Pipeline{Id: "pipeline-1", VariableDeclarations: `[{"name":"working_dir","value":"pipeline"},{"name":"working_dir","stage_id":"stage-frontend","value":"stage"}]`}
	stages := []model.StageDefinition{
		{Id: "stage-frontend", Name: "frontend", Script: `cd {{ working_dir | default: "web" }}`},
		{Id: "stage-backend", Name: "backend", Script: `cd {{ working_dir | default: "webapi" }}`},
	}

	_, values, err := ResolveRuntimeVariables(repo, pipeline, stages, RuntimeVariableOverrides{
		Global: map[string]string{"working_dir": "retry-global"},
		Stage:  map[string]map[string]string{"stage-frontend": {"working_dir": "retry-stage"}},
	})
	if err != nil {
		t.Fatalf("resolve runtime variables: %v", err)
	}
	if got := values.ValuesForStage(stages[0])["working_dir"]; got != "retry-stage" {
		t.Fatalf("frontend working_dir=%#v, want retry-stage", got)
	}
	if got := values.ValuesForStage(stages[1])["working_dir"]; got != "retry-global" {
		t.Fatalf("backend working_dir=%#v, want retry-global", got)
	}
}

func TestRuntimeVariableSnapshotRestoresGlobalAndStageScopedValues(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1", VariableDeclarations: `[{"name":"working_dir","value":"shared"},{"name":"working_dir","stage_id":"stage-frontend","value":"web-dist"}]`}
	stages := []model.StageDefinition{
		{Id: "stage-frontend", Name: "frontend", Script: `cd {{ working_dir | default: "web" }}`},
		{Id: "stage-backend", Name: "backend", Script: `cd {{ working_dir | default: "webapi" }}`},
	}

	declarations, runtime, err := ResolveRuntimeVariables(repo, pipeline, stages, RuntimeVariableOverrides{})
	if err != nil {
		t.Fatalf("resolve runtime variables: %v", err)
	}
	snapshot, err := MarshalRuntimeVariableSnapshot(runtime, declarations)
	if err != nil {
		t.Fatalf("marshal runtime variable snapshot: %v", err)
	}
	_, restored, err := UnmarshalRuntimeVariableSnapshot(snapshot)
	if err != nil {
		t.Fatalf("unmarshal runtime variable snapshot: %v", err)
	}
	if restored.Global["working_dir"] != "shared" {
		t.Fatalf("global values=%#v", restored.Global)
	}
	if got := restored.ValuesForStage(stages[0])["working_dir"]; got != "web-dist" {
		t.Fatalf("frontend working_dir=%#v, want web-dist", got)
	}
	if got := restored.ValuesForStage(stages[1])["working_dir"]; got != "shared" {
		t.Fatalf("backend working_dir=%#v, want shared", got)
	}
}

func TestResolveRuntimeVariablesRejectsUnknownStageScopedPipelineVariable(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1", VariableDeclarations: `[{"name":"working_dir","stage_id":"missing-stage","value":"web"}]`}
	stages := []model.StageDefinition{{Id: "stage-build", Name: "build", Script: "echo build"}}

	if _, _, err := ResolveRuntimeVariables(repo, pipeline, stages, RuntimeVariableOverrides{}); err == nil || !strings.Contains(err.Error(), "stage_id does not exist: missing-stage") {
		t.Fatalf("expected unknown stage scope error, got %v", err)
	}
}

func TestResolveRuntimeVariablesReportsRequiredVariableStage(t *testing.T) {
	repo := model.Repository{Id: "repo-1", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1"}
	stages := []model.StageDefinition{{Id: "stage-build", Name: "build", Script: `docker build -t {{ IMAGE_TAG }} .`}}

	_, _, err := ResolveRuntimeVariables(repo, pipeline, stages, RuntimeVariableOverrides{})
	if err == nil || !strings.Contains(err.Error(), "Stage build script") || !strings.Contains(err.Error(), "IMAGE_TAG") {
		t.Fatalf("expected stage-specific missing variable error, got %v", err)
	}
}

func TestResolvePipelineVariablesCollectsArtifactStageDefaults(t *testing.T) {
	artifacts := `[{"name":"{{ IMAGE_NAME | default: \"app\" }}","collector":"docker_image","reference":"registry/{{ IMAGE_NAME | default: \"app\" }}:{{ IMAGE_TAG | default: \"latest\" }}"}]`
	stages := []model.PipelineStage{{Id: "stage-build", Name: "build", Script: "echo build", Artifacts: &artifacts}}

	declarations, err := ResolvePipelineVariableDeclarations(stages, `[]`)
	if err != nil {
		t.Fatalf("resolve pipeline declarations: %v", err)
	}
	byName := make(map[string]model.VariableDeclaration, len(declarations))
	for _, declaration := range declarations {
		byName[declaration.Name] = declaration
	}
	if defaults := byName["IMAGE_NAME"].StageDefaults; len(defaults) != 2 || defaults[0].Default != "app" || defaults[1].Default != "app" {
		t.Fatalf("IMAGE_NAME defaults=%#v", defaults)
	}
	if defaults := byName["IMAGE_TAG"].StageDefaults; len(defaults) != 1 || defaults[0].Default != "latest" {
		t.Fatalf("IMAGE_TAG defaults=%#v", defaults)
	}
}

func TestNormalizePipelineVariablesRejectsInvalidNamesAndDuplicates(t *testing.T) {
	for _, variables := range [][]map[string]any{
		{{"name": "repository_ref"}},
		{{"name": "bad-name"}},
		{{"name": "IMAGE_TAG"}, {"name": "IMAGE_TAG"}},
		{{"name": "IMAGE_TAG", "stage_id": 1}},
	} {
		if _, err := NormalizePipelineVariables(variables); err == nil {
			t.Fatalf("variables %#v should be rejected", variables)
		}
	}
}

func TestNormalizePipelineVariablesDropsDerivedStageDefaults(t *testing.T) {
	variables, err := NormalizePipelineVariables([]map[string]any{{
		"name": "working_dir", "stage_defaults": []model.StageVariableDefault{{StageId: "stage-build", StageName: "build", Default: "backend"}},
	}})
	if err != nil {
		t.Fatalf("normalize pipeline variables: %v", err)
	}
	if _, exists := variables[0]["stage_defaults"]; exists {
		t.Fatalf("derived stage defaults must not be persisted: %#v", variables)
	}
}

func TestResolveTemplatePipelineVariablesPreservesTemplateDetailContract(t *testing.T) {
	stages := []model.PipelineStage{{Name: "build", Script: "echo {{ IMAGE_TAG }}"}}
	variables := []map[string]any{{"name": "invalid-name", "value": "allowed-for-template"}}

	resolved, err := ResolveTemplatePipelineVariables(stages, variables)
	if err != nil {
		t.Fatalf("resolve template variables: %v", err)
	}
	byName := make(map[string]map[string]any, len(resolved))
	for _, variable := range resolved {
		name, _ := variable["name"].(string)
		byName[name] = variable
	}
	if byName["IMAGE_TAG"]["source"] != "pipeline_stage" || byName["IMAGE_TAG"]["editable"] != false {
		t.Fatalf("stage declaration=%#v", byName["IMAGE_TAG"])
	}
	if _, exists := byName["repository_ref"]; !exists {
		t.Fatalf("template declarations=%#v", byName)
	}
	if byName["invalid-name"]["value"] != "allowed-for-template" {
		t.Fatalf("template custom declaration=%#v", byName["invalid-name"])
	}
}
