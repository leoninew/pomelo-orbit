package pipelinerunsvc

import (
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestSourceCommitArtifactForStage(t *testing.T) {
	clone := model.StageDefinition{
		Id:        "clone",
		Artifacts: []model.ArtifactConfig{{Collector: "command", Name: "source", Command: "git rev-parse HEAD", Format: "git_object_id"}},
	}
	build := model.StageDefinition{Id: "build", DependsOn: []string{"clone"}}
	artifact, err := sourceCommitArtifactForStage(build, map[string]model.StageDefinition{"clone": clone, "build": build})
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Artifact.Name != "source" || artifact.ProducerStageId != "clone" {
		t.Fatalf("unexpected source commit artifact: %+v", artifact)
	}
}

func TestBuildVersionLabelUsesRuntimeDatetime(t *testing.T) {
	if got := buildVersionLabel("20260805-142500"); got != "build-20260805-142500" {
		t.Fatalf("unexpected build version label: %q", got)
	}
}
