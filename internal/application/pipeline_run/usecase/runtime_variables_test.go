package pipelinerunsvc

import (
	"strings"
	"testing"

	pipelinevariable "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/rule/pipelinevariable"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestBuildPipelineRunVariablesIgnoresSnapshotVariableValues(t *testing.T) {
	repo := model.Repository{Id: "repo-1", Name: "Repository", Code: "repo", RepositoryUrl: "https://example.invalid/repo.git", DefaultBranch: "main"}
	pipeline := model.Pipeline{Id: "pipeline-1", Name: "Pipeline", Version: 4, VariableDeclarations: `[{"name":"IMAGE_TAG","default":"current-pipeline"}]`}
	snapshot := model.PipelineSnapshot{
		StagesSnapshot:    `[{"name":"build","script":"echo {{ IMAGE_TAG | default: \"stage\" }}"}]`,
		VariablesSnapshot: `[{"name":"IMAGE_TAG","value":"stale-snapshot","source":"pipeline","editable":true}]`,
	}

	data, ref, err := buildPipelineRunVariables(repo, pipeline, snapshot, map[string]string{
		"repository_ref": "release",
		"IMAGE_TAG":      "submitted",
	})
	if err != nil {
		t.Fatalf("build run variables: %v", err)
	}
	_, values, err := pipelinevariable.UnmarshalRuntimeVariableSnapshot(data)
	if err != nil {
		t.Fatalf("decode run variable snapshot: %v", err)
	}
	if ref != "release" || values["repository_ref"] != "release" || values["IMAGE_TAG"] != "submitted" {
		t.Fatalf("run values=%#v, ref=%q", values, ref)
	}
}

func TestPipelineRunExecutionVariablesRequiresMatchingRepositoryRef(t *testing.T) {
	variables := []model.VariableDeclaration{
		{Name: "repository_ref", Value: "release", Source: "runtime", Editable: true},
		{Name: "repository_code", Value: "repo", Source: "system"},
	}
	data, err := pipelinevariable.MarshalRuntimeVariableSnapshot(map[string]any{
		"repository_ref":  "release",
		"repository_code": "repo",
	}, variables)
	if err != nil {
		t.Fatalf("marshal run variable snapshot: %v", err)
	}

	service := Service{}
	if _, err := service.pipelineRunExecutionVariables(model.PipelineRun{RepositoryRef: "release", VariablesSnapshot: data}); err != nil {
		t.Fatalf("load matching run variables: %v", err)
	}
	if _, err := service.pipelineRunExecutionVariables(model.PipelineRun{RepositoryRef: "main", VariablesSnapshot: data}); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected mismatched ref error, got %v", err)
	}
}
