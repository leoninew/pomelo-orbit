package cisvc

import (
	"net/http"
	"testing"

	"gitee.com/leoninew/pomelo-orbit/internal/apperror"
	"gitee.com/leoninew/pomelo-orbit/internal/repository/model"
)

func TestNormalizeRepositoryCreateInputCompatibility(t *testing.T) {
	credential := " credential-1 "
	name, code, repositoryURL, defaultBranch, credentialId, err := normalizeRepositoryCreateInput(RepositoryCreateInput{
		Name:            " Repo One ",
		Code:            "repo-one",
		RepositoryURL:   " https://example.test/repo.git ",
		GitCredentialId: &credential,
	})
	if err != nil {
		t.Fatal(err)
	}
	if name != "Repo One" || code != "repo-one" || repositoryURL != "https://example.test/repo.git" || defaultBranch != "master" {
		t.Fatalf("unexpected normalized repository fields: name=%q code=%q url=%q branch=%q", name, code, repositoryURL, defaultBranch)
	}
	if credentialId == nil || *credentialId != "credential-1" {
		t.Fatalf("unexpected credential id: %v", credentialId)
	}

	_, _, _, _, _, err = normalizeRepositoryCreateInput(RepositoryCreateInput{Name: "Repo", Code: "Repo One", RepositoryURL: "https://example.test/repo.git"})
	if err == nil || apperror.StatusCode(err) != http.StatusBadRequest {
		t.Fatalf("expected invalid repository code to return 400, got %v", err)
	}
}

func TestRepositoryVariablesCompatibility(t *testing.T) {
	repo := model.Repository{Id: "repo-1", Name: "Repo One", Code: "repo-one", RepositoryURL: "https://example.test/repo.git", VariableOverrides: "", DefaultBranch: "main"}
	variables, err := repositoryVariables(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(variables) != 5 {
		t.Fatalf("expected repository builtins only, got %+v", variables)
	}
	if variables[0]["name"] != "repository_id" || variables[0]["default"] != "repo-1" || variables[0]["source"] != "repository" || variables[0]["editable"] != false {
		t.Fatalf("unexpected repository_id declaration: %+v", variables[0])
	}
	if variables[1]["name"] != "repository_name" || variables[1]["default"] != "Repo One" {
		t.Fatalf("unexpected repository_name declaration: %+v", variables[1])
	}
	if variables[4]["name"] != "repository_ref" || variables[4]["default"] != "main" || variables[4]["editable"] != true {
		t.Fatalf("unexpected repository_ref declaration: %+v", variables[4])
	}

	repo.VariableOverrides = `[{"name":"FOO","value":"bar"},{"name":"repository_id","value":"fake"}]`
	variables, err = repositoryVariables(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(variables) != 6 {
		t.Fatalf("expected builtins plus one custom variable, got %+v", variables)
	}
	custom := variables[5]
	if custom["name"] != "FOO" || custom["value"] != "bar" || custom["source"] != "repository_custom" || custom["editable"] != true {
		t.Fatalf("unexpected custom variable declaration: %+v", custom)
	}

	repo.VariableOverrides = `{`
	_, err = repositoryVariables(repo)
	if err == nil || apperror.StatusCode(err) != http.StatusInternalServerError {
		t.Fatalf("expected malformed stored variable_overrides to return 500, got %v", err)
	}
}

func TestWebhookBranchFilterCompatibility(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		filter string
		want   bool
	}{
		{name: "exact match", branch: "main", filter: "main", want: true},
		{name: "exact mismatch", branch: "develop", filter: "main", want: false},
		{name: "suffix wildcard", branch: "release/2026.06", filter: "release/*", want: true},
		{name: "prefix wildcard", branch: "feature/login", filter: "*/login", want: true},
		{name: "middle wildcard", branch: "hotfix/urgent/api", filter: "hotfix/*/api", want: true},
		{name: "wildcard mismatch", branch: "feature/api", filter: "release/*", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchBranchFilter(tt.branch, tt.filter); got != tt.want {
				t.Fatalf("matchBranchFilter(%q, %q) = %v, want %v", tt.branch, tt.filter, got, tt.want)
			}
		})
	}
}

func TestMarshalTriggerVariablesCompatibility(t *testing.T) {
	data, err := marshalTriggerVariables(nil)
	if err != nil {
		t.Fatal(err)
	}
	if data != "{}" {
		t.Fatalf("expected nil trigger variables to marshal as {}, got %s", data)
	}
}
