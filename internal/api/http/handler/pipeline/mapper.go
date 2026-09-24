package pipelinehandler

import (
	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"

	pipelinedto "github.com/leoninew/pomelo-orbit/internal/application/pipeline/dto"
	"github.com/leoninew/pomelo-orbit/internal/application/variableview"
	commonv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/common"
	pipelinev1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/pipeline"
)

func pipelineResponse(detail pipelinedto.PipelineDetail) *pipelinev1.PipelineResp {
	item := detail.Pipeline
	nodes := make([]*pipelinev1.PipelineStageNodeResp, 0, len(detail.StageNodes))
	for _, node := range detail.StageNodes {
		nodes = append(nodes, pipelineStageNodeResponse(node))
	}
	return &pipelinev1.PipelineResp{Id: item.Id, ProjectId: item.ProjectId, Kind: item.Kind, SourcePipelineId: item.SourcePipelineId, SourceTemplateName: item.SourceTemplateName, SourceTemplateVersion: transport.OptionalInt32(item.SourceTemplateVersion), ApplicationId: item.ApplicationId, ApplicationName: item.ApplicationName, RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, VersionForkStrategy: item.VersionForkStrategy, FixedVersionId: item.FixedVersionId, FixedVersionLabel: item.FixedVersionLabel, Name: item.Name, Description: item.Description, StageNodes: nodes, Variables: variableResponses(detail.Variables), Version: int32(item.Version), CreatedAt: transport.FormatTime(item.CreatedAt), UpdatedAt: transport.FormatTime(item.UpdatedAt)}
}

func variableResponses(items []variableview.View) []*commonv1.VariableResp {
	result := make([]*commonv1.VariableResp, 0, len(items))
	for _, item := range items {
		var binding *commonv1.VariableStageBindingResp
		if item.StageBinding != nil {
			binding = &commonv1.VariableStageBindingResp{StageId: item.StageBinding.StageId, StageName: item.StageBinding.StageName}
		}
		result = append(result, &commonv1.VariableResp{Name: item.Name, Kind: string(item.Kind), Scope: item.Scope, StageBinding: binding, Configuration: variableConfigurationResponse(item.Configuration), GlobalConfiguration: variableConfigurationResponse(item.GlobalConfiguration), StageOverride: variableConfigurationResponse(item.StageOverride), Editable: item.Editable})
	}
	return result
}

func variableConfigurationResponse(item *variableview.Configuration) *commonv1.VariableConfigurationResp {
	if item == nil {
		return nil
	}
	return &commonv1.VariableConfigurationResp{Description: item.Description, Default: transport.ProtoValue(item.Default), Value: transport.ProtoValue(item.Value), Secret: item.Secret, Editable: item.Editable}
}
func pipelineStageTemplateResponse(detail pipelinedto.PipelineStageTemplateDetail) *pipelinev1.PipelineStageResp {
	item := detail.Stage
	artifacts := make([]*pipelinev1.ArtifactConfigResp, 0, len(detail.Artifacts))
	for _, artifact := range detail.Artifacts {
		artifacts = append(artifacts, &pipelinev1.ArtifactConfigResp{Name: artifact.Name, Collector: artifact.Collector, Reference: artifact.Reference, Command: artifact.Command, Format: artifact.Format})
	}
	return &pipelinev1.PipelineStageResp{Id: item.Id, ProjectId: item.ProjectId, Kind: item.Kind, Name: item.Name, Image: item.Image, Script: item.Script, Description: item.Description, Version: int32(valueOrZero(item.Version)), CreatedAt: transport.FormatTime(item.CreatedAt), UpdatedAt: transport.FormatTime(item.UpdatedAt), Artifacts: artifacts}
}
func pipelineStageNodeResponse(detail pipelinedto.PipelineStageNodeDetail) *pipelinev1.PipelineStageNodeResp {
	item := detail.Node
	artifacts := make([]*pipelinev1.ArtifactConfigResp, 0, len(detail.Artifacts))
	for _, artifact := range detail.Artifacts {
		artifacts = append(artifacts, &pipelinev1.ArtifactConfigResp{Name: artifact.Name, Collector: artifact.Collector, Reference: artifact.Reference, Command: artifact.Command, Format: artifact.Format, ComponentName: artifact.ComponentName})
	}
	return &pipelinev1.PipelineStageNodeResp{Id: item.Id, NodeType: item.NodeType, PipelineId: item.PipelineId, Name: item.Name, Image: item.Image, Script: item.Script, Artifacts: artifacts, DependsOn: detail.DependsOn, SortOrder: int32(item.SortOrder), Description: item.Description, SourceTemplateStageId: item.SourceTemplateStageId, SourceTemplateStageName: item.SourceTemplateStageName, SourceTemplateStageDescription: item.SourceTemplateStageDescription, SourceTemplateStageVersion: int32(item.SourceTemplateStageVersion), LatestTemplateStageVersion: transport.OptionalInt32(detail.LatestTemplateStageVersion), CreatedAt: transport.FormatTime(item.CreatedAt), UpdatedAt: transport.FormatTime(item.UpdatedAt)}
}
func pipelineStageTemplateUpdatePreviewResponse(detail pipelinedto.PipelineStageTemplateUpdatePreview) *pipelinev1.PipelineStageTemplateUpdatePreviewResp {
	differences := make([]*pipelinev1.PipelineStageTemplateFieldDifferenceResp, 0, len(detail.Differences))
	for _, difference := range detail.Differences {
		differences = append(differences, &pipelinev1.PipelineStageTemplateFieldDifferenceResp{Field: difference.Field, Current: difference.Current, Target: difference.Target})
	}
	return &pipelinev1.PipelineStageTemplateUpdatePreviewResp{Available: detail.Available, Node: pipelineStageNodeResponse(detail.Node), ExpectedSourceTemplateStageVersion: int32(detail.ExpectedSourceTemplateVersion), TargetTemplateStageVersion: int32(detail.TargetTemplateVersion), Differences: differences}
}
func pipelineSnapshotResponse(detail pipelinedto.PipelineSnapshotDetail) *pipelinev1.PipelineSnapshotResp {
	item := detail.Snapshot
	stages := make([]*pipelinev1.SnapshotStageResp, 0, len(detail.StagesSnapshot))
	for _, stage := range detail.StagesSnapshot {
		artifacts := make([]*pipelinev1.ArtifactConfigResp, 0, len(stage.Artifacts))
		for _, artifact := range stage.Artifacts {
			artifacts = append(artifacts, &pipelinev1.ArtifactConfigResp{Name: artifact.Name, Collector: artifact.Collector, Reference: artifact.Reference, Command: artifact.Command, Format: artifact.Format, ComponentName: artifact.ComponentName})
		}
		stages = append(stages, &pipelinev1.SnapshotStageResp{Id: stage.Id, Name: stage.Name, Image: stage.Image, DependsOn: stage.DependsOn, Script: stage.Script, Artifacts: artifacts, SortOrder: int32(stage.SortOrder), Description: stage.Description})
	}
	return &pipelinev1.PipelineSnapshotResp{Id: item.Id, PipelineId: item.PipelineId, PipelineName: item.PipelineName, PipelineVersion: int32(item.PipelineVersion), SourcePipelineId: item.SourcePipelineId, SourceTemplateName: item.SourceTemplateName, SourceTemplateVersion: int32(item.SourceTemplateVersion), ApplicationId: item.ApplicationId, ApplicationName: item.ApplicationName, RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, VersionForkStrategy: item.VersionForkStrategy, FixedVersionId: item.FixedVersionId, FixedVersionLabel: item.FixedVersionLabel, StagesSnapshot: stages, Variables: variableResponses(detail.Variables), CreatedAt: transport.FormatTime(item.CreatedAt)}
}

func pipelineTemplateUpdatePreviewResponse(detail pipelinedto.PipelineTemplateUpdatePreview) *pipelinev1.PipelineTemplateUpdatePreviewResp {
	stages := make([]*pipelinev1.PipelineTemplateStageUpdateResp, 0, len(detail.Stages))
	for _, item := range detail.Stages {
		stages = append(stages, &pipelinev1.PipelineTemplateStageUpdateResp{StageId: item.StageId, SourceTemplateStageId: item.SourceTemplateStageId, Status: item.Status, CurrentVersion: int32(item.CurrentVersion), TargetVersion: int32(item.TargetVersion)})
	}
	variables := make([]*pipelinev1.PipelineTemplateVariableUpdateResp, 0, len(detail.Variables))
	for _, item := range detail.Variables {
		variables = append(variables, &pipelinev1.PipelineTemplateVariableUpdateResp{Name: item.Name, StageId: item.StageId, Status: item.Status, Current: item.Current, Target: item.Target})
	}
	return &pipelinev1.PipelineTemplateUpdatePreviewResp{Available: detail.Available, ExpectedPipelineVersion: int32(detail.ExpectedPipelineVersion), ExpectedSourceTemplateVersion: int32(detail.ExpectedSourceTemplateVersion), TargetSourceTemplateVersion: int32(detail.TargetSourceTemplateVersion), SourceTemplateName: detail.SourceTemplateName, Stages: stages, Variables: variables, Conflicts: detail.Conflicts}
}
func serviceArtifacts(items []*pipelinev1.ArtifactConfigReq) []pipelinedto.ArtifactConfig {
	result := make([]pipelinedto.ArtifactConfig, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, pipelinedto.ArtifactConfig{Name: item.Name, Collector: item.Collector, Reference: item.Reference, Command: item.Command, Format: item.Format, ComponentName: item.ComponentName})
		}
	}
	return result
}
func variableRequestMaps(items []*commonv1.VariableDeclarationReq) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, map[string]any{"name": item.Name, "description": item.Description, "default": transport.NativeValue(item.Default), "value": transport.NativeValue(item.Value), "secret": item.Secret, "source": item.Source, "editable": item.Editable, "stage_id": item.StageId})
		}
	}
	return result
}
func int32PtrToInt(value *int32) *int {
	if value == nil {
		return nil
	}
	result := int(*value)
	return &result
}
func valueOrZero(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
