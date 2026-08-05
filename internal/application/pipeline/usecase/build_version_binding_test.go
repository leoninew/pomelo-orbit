package pipelinesvc

import (
	"strings"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestValidateSourceCommitDependencies(t *testing.T) {
	clone := model.StageDefinition{
		Id: "clone", Name: "clone",
		Artifacts: []model.ArtifactConfig{{Collector: "command", Name: "source", Command: "git rev-parse HEAD", Format: "git_object_id"}},
	}
	build := model.StageDefinition{
		Id: "build", Name: "build", DependsOn: []string{"clone"},
		Artifacts: []model.ArtifactConfig{{Collector: "docker_image", Name: "image", Reference: "demo:latest"}},
	}
	if err := validateSourceCommitDependencies([]model.StageDefinition{clone, build}); err != nil {
		t.Fatalf("expected valid source commit dependency: %v", err)
	}

	build.DependsOn = nil
	err := validateSourceCommitDependencies([]model.StageDefinition{clone, build})
	if err == nil || !strings.Contains(err.Error(), "found 0") {
		t.Fatalf("expected missing source commit validation error, got %v", err)
	}
}

func TestValidateSourceCommitDependenciesRejectsMultipleProducers(t *testing.T) {
	first := model.StageDefinition{
		Id: "clone-a", Name: "clone-a",
		Artifacts: []model.ArtifactConfig{{Collector: "command", Name: "source-a", Command: "git rev-parse HEAD", Format: "git_object_id"}},
	}
	second := model.StageDefinition{
		Id: "clone-b", Name: "clone-b",
		Artifacts: []model.ArtifactConfig{{Collector: "command", Name: "source-b", Command: "git rev-parse HEAD", Format: "git_object_id"}},
	}
	build := model.StageDefinition{
		Id: "build", Name: "build", DependsOn: []string{"clone-a", "clone-b"},
		Artifacts: []model.ArtifactConfig{{Collector: "docker_image", Name: "image", Reference: "demo:latest"}},
	}
	err := validateSourceCommitDependencies([]model.StageDefinition{first, second, build})
	if err == nil || !strings.Contains(err.Error(), "found 2") {
		t.Fatalf("expected multiple source commit validation error, got %v", err)
	}
}
