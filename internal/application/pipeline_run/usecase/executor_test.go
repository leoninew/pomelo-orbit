package pipelinerunsvc

import (
	"context"
	"testing"

	applicationport "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/port"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
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
