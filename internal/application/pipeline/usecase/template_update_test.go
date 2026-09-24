package pipelinesvc

import (
	"encoding/json"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestMergeApplicationStagesUsesSnapshotForSameVersionContentChanges(t *testing.T) {
	artifacts := `[{"name":"image","collector":"docker_image","reference":"registry.example/old:latest"}]`
	updatedArtifacts := `[{"name":"image","collector":"docker_image","reference":"registry.example/new:latest"}]`
	dependsOn := "[]"
	sortOrder := 0
	version := 1
	sourceId, sourceName, sourceDescription := "stage-template", "build", "build template"
	current := []model.PipelineStage{{
		Id: "application-stage", ProjectId: "project-1", Kind: model.PipelineStageKindApplication,
		Name: "private build", Image: "old-image", Script: "old-script", Description: "private",
		SourceTemplateStageId: &sourceId, SourceTemplateStageName: &sourceName, SourceTemplateStageVersion: &version,
		SourceTemplateStageDescription: &sourceDescription, Artifacts: &artifacts, DependsOn: &dependsOn, SortOrder: sortOrder,
	}}
	references := []model.PipelineStageReference{{
		Id: "reference-1", SourceTemplateStageId: sourceId, SourceTemplateStageName: sourceName,
		SourceTemplateStageVersion: version, SourceTemplateStageDescription: sourceDescription,
		Name: "build", Image: "new-image", Script: "new-script", Description: "template description",
		Artifacts: updatedArtifacts, DependsOn: dependsOn, SortOrder: sortOrder,
	}}
	old := []model.StageDefinition{{
		Id: "reference-1", Name: "build", Image: "old-image", Script: "old-script", Description: "template description",
		Artifacts:             []model.ArtifactConfig{{Name: "image", Collector: "docker_image", Reference: "registry.example/old:latest"}},
		SourceTemplateStageId: sourceId, SourceTemplateStageName: sourceName, SourceTemplateStageVersion: version,
		SourceTemplateStageDescription: sourceDescription,
	}}
	merged, updates, conflicts, err := mergeApplicationStages(current, references, old)
	if err != nil {
		t.Fatalf("merge stages: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %v", conflicts)
	}
	if len(merged) != 1 || merged[0].Image != "new-image" || merged[0].Script != "new-script" || merged[0].Name != "private build" {
		t.Fatalf("merged stage = %#v", merged)
	}
	if len(updates) != 1 || updates[0].Status != pipelineTemplateUpdateStatusUpdate {
		t.Fatalf("stage updates = %#v", updates)
	}
}

func TestMergeApplicationStagesPreservesComponentMappings(t *testing.T) {
	componentName := "api"
	currentArtifacts := `[ {"name":"image","collector":"docker_image","reference":"registry.example/old:latest","component_name":"api"} ]`
	templateArtifacts := `[ {"name":"image","collector":"docker_image","reference":"registry.example/new:latest"} ]`
	dependsOn := "[]"
	sortOrder := 0
	version := 1
	sourceId, sourceName, sourceDescription := "stage-template", "build", "build template"
	current := []model.PipelineStage{{
		Id: "application-stage", ProjectId: "project-1", Kind: model.PipelineStageKindApplication,
		Name: "private build", Image: "old-image", Script: "old-script", Description: "private",
		SourceTemplateStageId: &sourceId, SourceTemplateStageName: &sourceName, SourceTemplateStageVersion: &version,
		SourceTemplateStageDescription: &sourceDescription, Artifacts: &currentArtifacts, DependsOn: &dependsOn, SortOrder: sortOrder,
	}}
	references := []model.PipelineStageReference{{
		Id: "reference-1", SourceTemplateStageId: sourceId, SourceTemplateStageName: sourceName,
		SourceTemplateStageVersion: 2, SourceTemplateStageDescription: sourceDescription,
		Name: "build", Image: "new-image", Script: "new-script", Description: "template description",
		Artifacts: templateArtifacts, DependsOn: dependsOn, SortOrder: sortOrder,
	}}
	old := []model.StageDefinition{{
		Id: "reference-1", Name: "build", Image: "old-image", Script: "old-script", Description: "template description",
		Artifacts:             []model.ArtifactConfig{{Name: "image", Collector: "docker_image", Reference: "registry.example/old:latest"}},
		SourceTemplateStageId: sourceId, SourceTemplateStageName: sourceName, SourceTemplateStageVersion: version,
		SourceTemplateStageDescription: sourceDescription,
	}}
	merged, _, conflicts, err := mergeApplicationStages(current, references, old)
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("merge stages: merged=%#v conflicts=%v err=%v", merged, conflicts, err)
	}
	if len(merged) != 1 || merged[0].Artifacts == nil {
		t.Fatalf("merged stages = %#v", merged)
	}
	artifacts, err := pipelineStageArtifacts(merged[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 || artifacts[0].ComponentName == nil || *artifacts[0].ComponentName != componentName || artifacts[0].Reference != "registry.example/new:latest" {
		t.Fatalf("merged artifacts = %#v", artifacts)
	}
}

func TestMergeApplicationVariablesPreservesChangedApplicationValue(t *testing.T) {
	oldJSON := `[ {"name":"image_tag","stage_id":"reference-1","value":"old","default":"old"} ]`
	currentJSON := `[ {"name":"image_tag","stage_id":"application-stage","value":"private","default":"old"} ]`
	targetJSON := `[ {"name":"image_tag","stage_id":"reference-1","value":"new","default":"new"} ]`
	version := 1
	sourceId := "stage-template"
	stages := []model.PipelineStage{{Id: "application-stage", SourceTemplateStageId: &sourceId, SourceTemplateStageVersion: &version}}
	references := []model.PipelineStageReference{{Id: "reference-1", SourceTemplateStageId: sourceId}}
	oldVariables := mustVariables(t, oldJSON)
	variables, updates, conflicts, err := mergeApplicationVariables(currentJSON, targetJSON, oldVariables, []model.StageDefinition{{Id: "reference-1", SourceTemplateStageId: sourceId}}, references, stages)
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("merge variables: variables=%s conflicts=%v err=%v", variables, conflicts, err)
	}
	var merged []model.VariableDeclaration
	if err := json.Unmarshal([]byte(variables), &merged); err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 || merged[0].Value != "private" || len(updates) != 1 || updates[0].Status != pipelineTemplateUpdateStatusKeep {
		t.Fatalf("merged variables=%#v updates=%#v", merged, updates)
	}
}

func TestMergeApplicationVariablesFollowsTemplateWhenApplicationValueUnchanged(t *testing.T) {
	oldJSON := `[ {"name":"image_tag","value":"old","default":"old"} ]`
	currentJSON := `[ {"name":"image_tag","value":"old","default":"old"} ]`
	targetJSON := `[ {"name":"image_tag","value":"new","default":"new"} ]`
	oldVariables := mustVariables(t, oldJSON)
	variables, updates, conflicts, err := mergeApplicationVariables(currentJSON, targetJSON, oldVariables, nil, nil, nil)
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("merge variables: variables=%s conflicts=%v err=%v", variables, conflicts, err)
	}
	var merged []model.VariableDeclaration
	if err := json.Unmarshal([]byte(variables), &merged); err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 || merged[0].Value != "new" || len(updates) != 1 || updates[0].Status != pipelineTemplateUpdateStatusUpdate {
		t.Fatalf("merged variables=%#v updates=%#v", merged, updates)
	}
}

func mustVariables(t *testing.T, value string) []model.VariableDeclaration {
	t.Helper()
	var result []model.VariableDeclaration
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		t.Fatal(err)
	}
	return result
}
