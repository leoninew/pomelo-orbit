package pipelinesvc

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestClonePipelineStagesRemapsDependenciesAndClearsComponentMappings(t *testing.T) {
	t.Parallel()

	componentName := "api"
	artifacts, err := json.Marshal([]model.ArtifactConfig{{
		Name:          "image",
		Collector:     "docker_image",
		Reference:     "registry.example/api:build",
		ComponentName: &componentName,
	}})
	if err != nil {
		t.Fatalf("marshal artifacts: %v", err)
	}
	dependsOn, err := json.Marshal([]string{"stage-source"})
	if err != nil {
		t.Fatalf("marshal dependencies: %v", err)
	}

	cloned, err := clonePipelineStages([]model.PipelineStage{
		{Id: "stage-source", PipelineId: "template", Name: "source", DependsOn: "[]", SortOrder: 1},
		{Id: "stage-build", PipelineId: "template", Name: "build", Artifacts: stringPointer(string(artifacts)), DependsOn: string(dependsOn), SortOrder: 2},
	}, "application-pipeline")
	if err != nil {
		t.Fatalf("clone stages: %v", err)
	}
	if len(cloned) != 2 {
		t.Fatalf("expected 2 cloned stages, got %d", len(cloned))
	}
	if cloned[0].PipelineId != "application-pipeline" || cloned[1].PipelineId != "application-pipeline" {
		t.Fatal("cloned stages must belong to the application pipeline")
	}
	if cloned[0].Id == "stage-source" || cloned[1].Id == "stage-build" || cloned[0].Id == cloned[1].Id {
		t.Fatal("cloned stages must receive distinct new ids")
	}

	var clonedDependencies []string
	if err := json.Unmarshal([]byte(cloned[1].DependsOn), &clonedDependencies); err != nil {
		t.Fatalf("decode cloned dependencies: %v", err)
	}
	if len(clonedDependencies) != 1 || clonedDependencies[0] != cloned[0].Id {
		t.Fatalf("expected dependency to be remapped to %q, got %v", cloned[0].Id, clonedDependencies)
	}

	var clonedArtifacts []model.ArtifactConfig
	if cloned[1].Artifacts == nil {
		t.Fatal("expected cloned artifacts")
	}
	if err := json.Unmarshal([]byte(*cloned[1].Artifacts), &clonedArtifacts); err != nil {
		t.Fatalf("decode cloned artifacts: %v", err)
	}
	if clonedArtifacts[0].ComponentName != nil {
		t.Fatalf("template component mapping leaked into application pipeline: %q", *clonedArtifacts[0].ComponentName)
	}
}

func TestGetOrCreatePipelineSnapshotRejectsTemplate(t *testing.T) {
	t.Parallel()

	_, err := GetOrCreatePipelineSnapshot(context.Background(), nil, model.Pipeline{Kind: model.PipelineKindTemplate})
	if err == nil || !strings.Contains(err.Error(), "template pipelines cannot create snapshots") {
		t.Fatalf("expected template snapshot rejection, got %v", err)
	}
}

func TestValidatePipelineConfigurationAllowsApplicationPipelineWithoutApplicationBinding(t *testing.T) {
	t.Parallel()

	sourcePipelineID, sourceTemplateName := "template-1", "Build template"
	sourceTemplateVersion := 1
	repositoryID, repositoryName := "repository-1", "source"
	pipeline := model.Pipeline{
		Kind:                  model.PipelineKindApplication,
		SourcePipelineId:      &sourcePipelineID,
		SourceTemplateName:    &sourceTemplateName,
		SourceTemplateVersion: &sourceTemplateVersion,
		RepositoryId:          &repositoryID,
		RepositoryName:        &repositoryName,
	}

	if err := (Service{}).validatePipelineConfiguration(context.Background(), pipeline, nil); err != nil {
		t.Fatalf("expected unbound application pipeline to be valid, got %v", err)
	}
}

func TestValidatePipelineConfigurationRejectsComponentMappingWithoutApplicationBinding(t *testing.T) {
	t.Parallel()

	sourcePipelineID, sourceTemplateName := "template-1", "Build template"
	sourceTemplateVersion := 1
	repositoryID, repositoryName := "repository-1", "source"
	componentName := "api"
	artifacts, err := json.Marshal([]model.ArtifactConfig{{
		Name:          "image",
		Collector:     "docker_image",
		Reference:     "registry.example/api:build",
		ComponentName: &componentName,
	}})
	if err != nil {
		t.Fatalf("marshal artifacts: %v", err)
	}
	pipeline := model.Pipeline{
		Kind:                  model.PipelineKindApplication,
		SourcePipelineId:      &sourcePipelineID,
		SourceTemplateName:    &sourceTemplateName,
		SourceTemplateVersion: &sourceTemplateVersion,
		RepositoryId:          &repositoryID,
		RepositoryName:        &repositoryName,
	}
	stages := []model.PipelineStage{{
		Id: "stage-build", Name: "build", Image: "builder", Script: "build", Artifacts: stringPointer(string(artifacts)), DependsOn: "[]",
	}}

	err = (Service{}).validatePipelineConfiguration(context.Background(), pipeline, stages)
	if err == nil || !strings.Contains(err.Error(), "require an application binding") {
		t.Fatalf("expected application binding rejection, got %v", err)
	}
}

func TestSourceCommitArtifactForStageUsesTransitiveDependency(t *testing.T) {
	t.Parallel()

	componentName := "api"
	artifact, err := sourceCommitArtifactForStage(model.StageDefinition{
		Id:        "build",
		Name:      "build",
		DependsOn: []string{"source"},
		Artifacts: []model.ArtifactConfig{{Name: "image", Collector: "docker_image", ComponentName: &componentName}},
	}, []model.StageDefinition{
		{Id: "source", Name: "source", Artifacts: []model.ArtifactConfig{{Name: "commit", Collector: "command", Command: "git rev-parse HEAD", Format: "git_object_id"}}},
		{Id: "build", Name: "build", DependsOn: []string{"source"}},
	})
	if err != nil {
		t.Fatalf("resolve source commit artifact: %v", err)
	}
	if artifact.Name != "commit" {
		t.Fatalf("expected commit artifact, got %q", artifact.Name)
	}
}

func stringPointer(value string) *string { return &value }
