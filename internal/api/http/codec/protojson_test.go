package codec

import (
	"strings"
	"testing"

	repositoryv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/repository"
)

func TestMarshalProtoJSONUsesProtoNames(t *testing.T) {
	data, err := MarshalProtoJSON(&repositoryv1.RepositoryResp{RepositoryUrl: "https://example.test/repo.git"})
	if err != nil {
		t.Fatalf("MarshalProtoJSON() error = %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `"repository_url"`) {
		t.Fatalf("MarshalProtoJSON() = %s, want repository_url field", text)
	}
	if strings.Contains(text, `"repositoryUrl"`) {
		t.Fatalf("MarshalProtoJSON() = %s, must not use repositoryUrl field", text)
	}
}

func TestMarshalProtoJSONEmitsUnpopulatedFields(t *testing.T) {
	data, err := MarshalProtoJSON(&repositoryv1.RepositoryResp{})
	if err != nil {
		t.Fatalf("MarshalProtoJSON() error = %v", err)
	}
	text := string(data)
	for _, field := range []string{`"has_credential":false`, `"git_credential_id":""`, `"variable_declarations":[]`} {
		if !strings.Contains(text, field) {
			t.Fatalf("MarshalProtoJSON() = %s, want %s", text, field)
		}
	}
}

func TestUnmarshalProtoJSONRejectsUnknownFields(t *testing.T) {
	var req repositoryv1.RepositoryCreateReq
	err := UnmarshalProtoJSON([]byte(`{"name":"repo","unknown_field":"x"}`), &req)
	if err == nil {
		t.Fatal("UnmarshalProtoJSON() error = nil, want unknown field error")
	}
}
