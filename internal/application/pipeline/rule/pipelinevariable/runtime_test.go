package pipelinevariable

import (
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestBuildRuntimeVariablesPreservesCustomOverridePrecedence(t *testing.T) {
	declarations := []model.VariableDeclaration{
		{Name: "working_dir", Default: ".", Source: "pipeline_custom", Editable: true},
		{Name: "repository_dockerfile", Default: "Dockerfile", Source: "pipeline_custom", Editable: true},
	}
	repository := model.Repository{
		Id: "repository-1", Name: "repository", Code: "repository", RepositoryUrl: "https://example.invalid/repository.git",
		VariableOverrides: `[{"name":"repository_dockerfile","value":"Containerfile","source":"repository_custom","editable":true}]`,
	}
	pipeline := model.Pipeline{Id: "pipeline-1", Name: "pipeline", Version: 1}

	variables, err := BuildRuntimeVariables(repository, pipeline, "main", nil, declarations)
	if err != nil {
		t.Fatalf("BuildRuntimeVariables returned error: %v", err)
	}
	if got := variables["working_dir"]; got != "." {
		t.Fatalf("working_dir = %#v, want .", got)
	}
	if got := variables["repository_dockerfile"]; got != "Containerfile" {
		t.Fatalf("repository_dockerfile = %#v, want Containerfile", got)
	}

	variables, err = BuildRuntimeVariables(repository, pipeline, "main", map[string]string{
		"repository_dockerfile": "Dockerfile.release",
		"repository_ref":        "cannot-override-builtin",
	}, declarations)
	if err != nil {
		t.Fatalf("BuildRuntimeVariables with overrides returned error: %v", err)
	}
	if got := variables["repository_dockerfile"]; got != "Dockerfile.release" {
		t.Fatalf("repository_dockerfile = %#v, want Dockerfile.release", got)
	}
	if got := variables["repository_ref"]; got != "main" {
		t.Fatalf("repository_ref = %#v, want main", got)
	}
}

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
