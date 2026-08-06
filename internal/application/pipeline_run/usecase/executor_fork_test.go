package pipelinerunsvc

import (
	"context"
	"testing"

	applicationport "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/port"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestForkBuildVersionDelegatesToApplicationVersionForker(t *testing.T) {
	image := "example/web:build"
	store := &fakeBuildArtifactStore{
		fakeExecutionStore: &fakeExecutionStore{},
		binding: model.PipelineRunBuildVersionBinding{
			SourceVersionId: "version-source",
			ComponentName:   "web",
		},
	}
	forker := &fakeBuildVersionForker{version: model.Version{Id: "version-generated", Label: "build-20260806-120000"}}
	executor := Executor{store: store, versionForker: forker}

	err := executor.forkBuildVersion(
		context.Background(),
		store,
		model.PipelineRun{Id: "run-1"},
		model.StageDefinition{Id: "stage-1"},
		model.Artifact{Id: "artifact-1", ImageRef: &image},
		"20260806-120000",
	)
	if err != nil {
		t.Fatal(err)
	}
	if forker.input.SourceVersionId != "version-source" || forker.input.Label != "build-20260806-120000" || forker.input.ComponentName != "web" || forker.input.Image != image || forker.input.ArtifactId != "artifact-1" {
		t.Fatalf("unexpected fork request: %+v", forker)
	}
	if store.generatedVersionId != "version-generated" || store.generatedVersionLabel != "build-20260806-120000" || store.completedArtifactId != "artifact-1" {
		t.Fatalf("unexpected completed binding: %+v", store)
	}
}

type fakeBuildArtifactStore struct {
	*fakeExecutionStore
	binding               model.PipelineRunBuildVersionBinding
	generatedVersionId    string
	generatedVersionLabel string
	completedArtifactId   string
}

func (s *fakeBuildArtifactStore) PipelineRunBuildVersionBinding(_ context.Context, _ string, _ string) (model.PipelineRunBuildVersionBinding, error) {
	return s.binding, nil
}

func (s *fakeBuildArtifactStore) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (s *fakeBuildArtifactStore) CompletePipelineRunBuildVersionBinding(_ context.Context, _ string, _ string, versionId string, label string, artifactId string) error {
	s.generatedVersionId = versionId
	s.generatedVersionLabel = label
	s.completedArtifactId = artifactId
	return nil
}

type fakeBuildVersionForker struct {
	version model.Version
	input   applicationport.BuildVersionForkInput
}

func (s *fakeBuildVersionForker) ForkVersionForBuild(_ context.Context, input applicationport.BuildVersionForkInput) (model.Version, error) {
	s.input = input
	return s.version, nil
}
