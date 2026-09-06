package pipelinerunsvc

import (
	"context"
	"strings"
	"testing"

	applicationport "github.com/leoninew/pomelo-orbit/internal/application/application/port"
	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestForkBuildVersionCompletesBindingInForkTransaction(t *testing.T) {
	componentName := "api"
	store := &forkBuildVersionStoreStub{
		binding: model.PipelineRunVersionBinding{SourceVersionId: "version-source"},
		artifacts: []model.Artifact{{
			Id: "artifact-1", PipelineStageId: "stage-1", Name: "image",
			ImageRef: stringRef("registry.example/api:build"), LocalImageSha256: stringRef("sha256:abc"), SourceCommitSha: stringRef("0123456789012345678901234567890123456789"),
		}},
	}
	transactionRunner := &transactionRunnerStub{}
	forker := &buildVersionForkerStub{}
	err := forkBuildVersion(context.Background(), store, transactionRunner, forker, model.PipelineRun{Id: "run-1"}, []model.StageDefinition{{
		Id: "stage-1", Name: "Build", Artifacts: []model.ArtifactConfig{{Name: "image", Collector: "docker_image", ComponentName: &componentName}},
	}}, "20260807170000")
	if err != nil {
		t.Fatal(err)
	}
	if !store.completedInTransaction || !forker.calledInTransaction {
		t.Fatal("version fork and binding completion must use the same transaction context")
	}
	if store.generatedVersionID != "version-generated" || store.generatedVersionLabel != "build-20260807170000" {
		t.Fatalf("binding=%q/%q", store.generatedVersionID, store.generatedVersionLabel)
	}
	if len(forker.input.Components) != 1 || forker.input.Components[0].ComponentName != "api" {
		t.Fatalf("fork input=%+v", forker.input)
	}
}

func TestValidateArtifactPayload(t *testing.T) {
	tests := []struct {
		name     string
		artifact model.Artifact
		wantErr  bool
	}{
		{
			name:     "file",
			artifact: model.Artifact{Collector: "file", Location: stringRef("/data/report.txt")},
		},
		{
			name: "command",
			artifact: model.Artifact{
				Collector: "command", Value: stringRef("abc"), ValueFormat: stringRef("text"),
			},
		},
		{
			name: "docker image",
			artifact: model.Artifact{
				Collector: "docker_image", ImageRef: stringRef("registry.example/api:build"), LocalImageSha256: stringRef("sha256:abc"),
			},
		},
		{
			name: "file cannot include a value",
			artifact: model.Artifact{
				Collector: "file", Location: stringRef("/data/report.txt"), Value: stringRef("unexpected"),
			},
			wantErr: true,
		},
		{
			name: "command requires a supported value format",
			artifact: model.Artifact{
				Collector: "command", Value: stringRef("abc"), ValueFormat: stringRef("json"),
			},
			wantErr: true,
		},
		{
			name:     "docker image requires its digest",
			artifact: model.Artifact{Collector: "docker_image", ImageRef: stringRef("registry.example/api:build")},
			wantErr:  true,
		},
		{
			name:     "unknown collector",
			artifact: model.Artifact{Collector: "archive"},
			wantErr:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateArtifactPayload(test.artifact)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateArtifactPayload() error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}
}

type transactionContextKey struct{}

type forkBuildVersionStoreStub struct {
	binding                model.PipelineRunVersionBinding
	artifacts              []model.Artifact
	completedInTransaction bool
	generatedVersionID     string
	generatedVersionLabel  string
}

func (s *forkBuildVersionStoreStub) PipelineRunVersionBinding(context.Context, string) (model.PipelineRunVersionBinding, error) {
	return s.binding, nil
}

func (s *forkBuildVersionStoreStub) ListArtifactsByRun(context.Context, *string, string) ([]model.Artifact, error) {
	return s.artifacts, nil
}

func (s *forkBuildVersionStoreStub) CompletePipelineRunVersionBinding(ctx context.Context, _ string, versionID string, label string) error {
	s.completedInTransaction, _ = ctx.Value(transactionContextKey{}).(bool)
	s.generatedVersionID, s.generatedVersionLabel = versionID, label
	return nil
}

type transactionRunnerStub struct{}

func (s *transactionRunnerStub) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(context.WithValue(ctx, transactionContextKey{}, true))
}

type buildVersionForkerStub struct {
	calledInTransaction bool
	input               applicationport.BuildVersionForkInput
}

func (s *buildVersionForkerStub) ForkVersionForBuild(ctx context.Context, input applicationport.BuildVersionForkInput) (model.Version, error) {
	s.calledInTransaction, _ = ctx.Value(transactionContextKey{}).(bool)
	s.input = input
	return model.Version{Id: "version-generated", Label: input.Label}, nil
}

func stringRef(value string) *string { return &value }

func TestAuthenticatedRepositoryUrlRewritesGiteaToken(t *testing.T) {
	const secretKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	encrypted, err := security.EncryptString(secretKey, "alice:gitea-access-token")
	if err != nil {
		t.Fatal(err)
	}
	executor := Executor{secretKey: secretKey}
	credential := model.Credential{Type: "gitea_token", EncryptedData: encrypted}

	got, err := executor.authenticatedRepositoryUrl("https://git.example.com:3000/org/repo.git", credential)
	if err != nil {
		t.Fatal(err)
	}
	want := "https://alice:gitea-access-token@git.example.com:3000/org/repo.git"
	if got != want {
		t.Fatalf("url=%q want %q", got, want)
	}

	_, err = executor.authenticatedRepositoryUrl("https://git.example.com/org/repo.git", model.Credential{Type: "gitea_token", EncryptedData: mustEncrypt(t, secretKey, "gitea-access-token")})
	if err == nil || !strings.Contains(err.Error(), "gitea_token credential must be username:token") {
		t.Fatalf("expected username:token validation error, got %v", err)
	}
}

func mustEncrypt(t *testing.T, secretKey, plain string) string {
	t.Helper()
	encrypted, err := security.EncryptString(secretKey, plain)
	if err != nil {
		t.Fatal(err)
	}
	return encrypted
}
