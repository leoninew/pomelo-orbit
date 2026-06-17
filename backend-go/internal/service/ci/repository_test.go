package cisvc

import (
	"net/http"
	"testing"

	"backend/internal/apperror"
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
	variables, err := repositoryVariables("")
	if err != nil {
		t.Fatal(err)
	}
	if variables == nil || len(variables) != 0 {
		t.Fatalf("expected empty variable_overrides to decode as an empty slice, got %+v", variables)
	}

	variables, err = repositoryVariables(`[{"name":"FOO","value":"bar"}]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(variables) != 1 || variables[0]["name"] != "FOO" || variables[0]["value"] != "bar" {
		t.Fatalf("unexpected variable declarations: %+v", variables)
	}

	_, err = repositoryVariables(`{`)
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
