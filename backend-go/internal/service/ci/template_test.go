package cisvc

import (
	"testing"

	"backend/internal/repository/model"
)

func TestResolveStagesRendersScriptAndArtifacts(t *testing.T) {
	stages := []model.StageDefinition{{
		Id:     "stage-1",
		Name:   "build",
		Script: "docker build -t {{ IMAGE }} .",
		Artifacts: []model.ArtifactConfig{
			{Type: "docker_image", Name: "{{ IMAGE }}", Path: "{{ IMAGE }}:latest"},
		},
	}}

	resolved, err := resolveStages(stages, map[string]any{"IMAGE": "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if resolved[0].Script != "docker build -t demo ." {
		t.Fatalf("unexpected script: %s", resolved[0].Script)
	}
	if resolved[0].Artifacts[0].Name != "demo" || resolved[0].Artifacts[0].Path != "demo:latest" {
		t.Fatalf("unexpected artifact: %+v", resolved[0].Artifacts[0])
	}
}

func TestResolveStagesRejectsMissingVariable(t *testing.T) {
	_, err := resolveStages([]model.StageDefinition{{Id: "stage-1", Script: "{{ MISSING }}"}}, map[string]any{})
	if err == nil {
		t.Fatal("expected missing variable error")
	}
}
