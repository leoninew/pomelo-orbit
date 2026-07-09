package cisvc

import (
	"encoding/json"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestBuildPipelineRunVariablesMergesDefaultsAndProtectsBuiltins(t *testing.T) {
	repo := model.Repository{Id: "repo-1", Name: "Repo", Code: "real-code", RepositoryURL: "https://example.test/repo.git", VariableOverrides: `[{"name":"IMAGE","value":"repo-image","source":"repository_custom"}]`}
	template := model.PipelineTemplate{Id: "template-1", Name: "template", Version: 7, VariableDeclarations: `[{"name":"IMAGE","value":"template-image","source":"template_custom"}]`}
	snapshot := model.PipelineSnapshot{VariablesSnapshot: `[{"name":"repository_code","source":"template","editable":false},{"name":"IMAGE","default":"stage-image","source":"template_stage","editable":true},{"name":"working_dir","default":".","source":"template_stage","editable":true}]`}

	data, err := buildPipelineRunVariables(repo, template, snapshot, "main", map[string]string{"repository_code": "fake-code"})
	if err != nil {
		t.Fatal(err)
	}
	variables := declarationsByName(t, data)
	if variables["repository_code"].Value != "real-code" {
		t.Fatalf("builtin override was not ignored: %+v", variables["repository_code"])
	}
	if variables["IMAGE"].Value != "repo-image" {
		t.Fatalf("repository value should override template/stage defaults: %+v", variables["IMAGE"])
	}
	if variables["working_dir"].Value != "." {
		t.Fatalf("stage default was not used: %+v", variables["working_dir"])
	}
}

func TestBuildPipelineRunVariablesKeepsSecretSnapshotValue(t *testing.T) {
	repo := model.Repository{Id: "repo-1", Name: "Repo", Code: "repo", RepositoryURL: "https://example.test/repo.git"}
	template := model.PipelineTemplate{Id: "template-1", Name: "template", Version: 1, VariableDeclarations: `[{"name":"TOKEN","value":"secret-token","secret":true,"source":"template_custom"}]`}
	snapshot := model.PipelineSnapshot{VariablesSnapshot: `[{"name":"TOKEN","secret":true,"source":"template_custom"}]`}

	data, err := buildPipelineRunVariables(repo, template, snapshot, "main", nil)
	if err != nil {
		t.Fatal(err)
	}
	variables := declarationsByName(t, data)
	if variables["TOKEN"].Value != "secret-token" {
		t.Fatalf("secret value was not preserved: %+v", variables["TOKEN"])
	}
}

func declarationsByName(t *testing.T, value string) map[string]model.VariableDeclaration {
	t.Helper()
	var declarations []model.VariableDeclaration
	if err := json.Unmarshal([]byte(value), &declarations); err != nil {
		t.Fatal(err)
	}
	result := map[string]model.VariableDeclaration{}
	for _, declaration := range declarations {
		result[declaration.Name] = declaration
	}
	return result
}
