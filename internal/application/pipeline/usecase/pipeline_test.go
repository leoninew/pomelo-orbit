package pipelinesvc

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestClonePipelineStageReferencesRemapsDependenciesAndKeepsSourceSnapshots(t *testing.T) {
	t.Parallel()

	sourceArtifacts := `[{"name":"commit","collector":"command","command":"git rev-parse HEAD","format":"git_object_id"}]`
	buildArtifacts := `[{"name":"image","collector":"docker_image","reference":"registry.example/build:latest"}]`
	cloned, err := clonePipelineStageReferences([]model.PipelineStageReference{
		{Id: "stage-source", SourceTemplateStageId: "template-source", SourceTemplateStageName: "source", SourceTemplateStageVersion: 2, SourceTemplateStageDescription: "source template", Name: "source", Image: "alpine", Script: "source", Description: "local source", Artifacts: sourceArtifacts, DependsOn: "[]", SortOrder: 1},
		{Id: "stage-build", SourceTemplateStageId: "template-build", SourceTemplateStageName: "build", SourceTemplateStageVersion: 3, SourceTemplateStageDescription: "build template", Name: "build", Image: "builder", Script: "build", Description: "local build", Artifacts: buildArtifacts, DependsOn: "[\"stage-source\"]", SortOrder: 2},
	}, "application-pipeline", "project-1")
	if err != nil {
		t.Fatalf("clone stages: %v", err)
	}
	if len(cloned) != 2 {
		t.Fatalf("expected 2 cloned stages, got %d", len(cloned))
	}
	if cloned[0].PipelineId == nil || cloned[1].PipelineId == nil || *cloned[0].PipelineId != "application-pipeline" || *cloned[1].PipelineId != "application-pipeline" {
		t.Fatal("cloned stages must belong to the application pipeline")
	}
	if cloned[0].Id == "stage-source" || cloned[1].Id == "stage-build" || cloned[0].Id == cloned[1].Id {
		t.Fatal("cloned stages must receive distinct new ids")
	}
	if cloned[1].DependsOn == nil {
		t.Fatal("expected cloned dependencies")
	}
	var clonedDependencies []string
	if err := json.Unmarshal([]byte(*cloned[1].DependsOn), &clonedDependencies); err != nil {
		t.Fatalf("decode cloned dependencies: %v", err)
	}
	if len(clonedDependencies) != 1 || clonedDependencies[0] != cloned[0].Id {
		t.Fatalf("expected dependency to be remapped to %q, got %v", cloned[0].Id, clonedDependencies)
	}
	if cloned[1].Kind != model.PipelineStageKindApplication || cloned[1].SourceTemplateStageId == nil || *cloned[1].SourceTemplateStageId != "template-build" || cloned[1].SourceTemplateStageVersion == nil || *cloned[1].SourceTemplateStageVersion != 3 {
		t.Fatalf("source snapshot was not materialized: %#v", cloned[1])
	}
	if cloned[1].Artifacts == nil || *cloned[1].Artifacts != buildArtifacts {
		t.Fatalf("application stage must copy reference artifacts: %#v", cloned[1].Artifacts)
	}
}

func TestGetOrCreatePipelineSnapshotRejectsTemplate(t *testing.T) {
	t.Parallel()

	_, err := GetOrCreatePipelineSnapshot(context.Background(), nil, model.Pipeline{Kind: model.PipelineKindTemplate}, model.Repository{})
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
	pipelineID, sourceID, sourceName, sourceDescription, dependsOn := "pipeline-1", "template-stage-1", "build", "template description", "[]"
	sortOrder, sourceVersion := 0, 1
	stages := []model.PipelineStage{{
		Id: "stage-build", ProjectId: "project-1", Kind: model.PipelineStageKindApplication, PipelineId: &pipelineID, Name: "build", Image: "builder", Script: "build", Artifacts: stringPointer(string(artifacts)), DependsOn: &dependsOn, SortOrder: &sortOrder, SourceTemplateStageId: &sourceID, SourceTemplateStageName: &sourceName, SourceTemplateStageDescription: &sourceDescription, SourceTemplateStageVersion: &sourceVersion,
	}}

	err = (Service{}).validatePipelineConfiguration(context.Background(), pipeline, stages)
	if err == nil || !strings.Contains(err.Error(), "require an application binding") {
		t.Fatalf("expected application binding rejection, got %v", err)
	}
}

func TestPipelineStageDefinitionsRejectsInvalidApplicationStageShape(t *testing.T) {
	t.Parallel()

	pipelineID, sourceID, sourceName, sourceDescription, artifacts, dependsOn := "pipeline-1", "template-stage-1", "build", "template description", "[]", "[]"
	sortOrder, sourceVersion := 0, 0
	_, err := pipelineStageDefinitions([]model.PipelineStage{{
		Id: "stage-build", ProjectId: "project-1", Kind: model.PipelineStageKindApplication, PipelineId: &pipelineID,
		Name: "build", Image: "builder", Script: "build", Artifacts: &artifacts, DependsOn: &dependsOn, SortOrder: &sortOrder,
		SourceTemplateStageId: &sourceID, SourceTemplateStageName: &sourceName, SourceTemplateStageDescription: &sourceDescription, SourceTemplateStageVersion: &sourceVersion,
	}})
	if err == nil || !strings.Contains(err.Error(), "source snapshot is required") {
		t.Fatalf("expected business validation of application stage source snapshot, got %v", err)
	}
}

func TestValidateTemplatePipelineConfigurationRejectsInvalidReferenceShape(t *testing.T) {
	t.Parallel()

	err := validateTemplatePipelineConfiguration(model.Pipeline{Id: "template-pipeline", Kind: model.PipelineKindTemplate}, []model.PipelineStageReference{{
		Id: "reference-1", PipelineId: "template-pipeline", SourceTemplateStageId: "stage-template", SourceTemplateStageName: "build", SourceTemplateStageVersion: 0,
		Name: "build", Image: "alpine", Script: "true", DependsOn: "{}", SortOrder: -1,
	}})
	if err == nil || !strings.Contains(err.Error(), "invalid template pipeline stage reference") {
		t.Fatalf("expected business validation of template stage reference, got %v", err)
	}
}

func TestTemplateStageDeletionDependencyFindsDependentStage(t *testing.T) {
	t.Parallel()

	target, dependent, err := templateStageDeletionDependency([]model.PipelineStageReference{
		{Id: "source", Name: "source", DependsOn: "[]"},
		{Id: "build", Name: "build", DependsOn: `["source"]`},
	}, "source")
	if err != nil {
		t.Fatal(err)
	}
	if target != "source" || dependent != "build" {
		t.Fatalf("dependency = (%q, %q), want (source, build)", target, dependent)
	}
}

func TestApplicationStageDeletionDependencyFindsDependentStage(t *testing.T) {
	t.Parallel()

	sourceDependencies, buildDependencies := "[]", `["source"]`
	target, dependent, err := applicationStageDeletionDependency([]model.PipelineStage{
		{Id: "source", Name: "source", DependsOn: &sourceDependencies},
		{Id: "build", Name: "build", DependsOn: &buildDependencies},
	}, "source")
	if err != nil {
		t.Fatal(err)
	}
	if target != "source" || dependent != "build" {
		t.Fatalf("dependency = (%q, %q), want (source, build)", target, dependent)
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
