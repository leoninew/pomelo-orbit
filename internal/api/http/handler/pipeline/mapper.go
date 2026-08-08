package pipelinehandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	pipelinedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/dto"
	commonv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/common"
	pipelinev1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/pipeline"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func pipelineResponse(detail pipelinedto.PipelineDetail) *pipelinev1.PipelineResp {
	item := detail.Pipeline
	nodes := make([]*pipelinev1.PipelineStageNodeResp, 0, len(detail.StageNodes))
	for _, node := range detail.StageNodes {
		nodes = append(nodes, pipelineStageNodeResponse(node))
	}
	return &pipelinev1.PipelineResp{Id: item.Id, ProjectId: item.ProjectId, Kind: item.Kind, SourcePipelineId: item.SourcePipelineId, SourceTemplateName: item.SourceTemplateName, SourceTemplateVersion: intPtrToInt32(item.SourceTemplateVersion), ApplicationId: item.ApplicationId, ApplicationName: item.ApplicationName, RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, VersionForkStrategy: item.VersionForkStrategy, FixedVersionId: item.FixedVersionId, FixedVersionLabel: item.FixedVersionLabel, Name: item.Name, Description: item.Description, StageNodes: nodes, VariableDeclarations: variableResponses(detail.VariableDeclarations), Version: int32(item.Version), CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}
func pipelineStageTemplateResponse(detail pipelinedto.PipelineStageTemplateDetail) *pipelinev1.PipelineStageResp {
	item := detail.Stage
	artifacts := make([]*pipelinev1.ArtifactConfigResp, 0, len(detail.Artifacts))
	for _, artifact := range detail.Artifacts {
		artifacts = append(artifacts, &pipelinev1.ArtifactConfigResp{Name: artifact.Name, Collector: artifact.Collector, Reference: artifact.Reference, Command: artifact.Command, Format: artifact.Format})
	}
	return &pipelinev1.PipelineStageResp{Id: item.Id, ProjectId: item.ProjectId, Kind: item.Kind, Name: item.Name, Image: item.Image, Script: item.Script, Description: item.Description, Version: int32(valueOrZero(item.Version)), CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt), Artifacts: artifacts}
}
func pipelineStageNodeResponse(detail pipelinedto.PipelineStageNodeDetail) *pipelinev1.PipelineStageNodeResp {
	item := detail.Node
	artifacts := make([]*pipelinev1.ArtifactConfigResp, 0, len(detail.Artifacts))
	for _, artifact := range detail.Artifacts {
		artifacts = append(artifacts, &pipelinev1.ArtifactConfigResp{Name: artifact.Name, Collector: artifact.Collector, Reference: artifact.Reference, Command: artifact.Command, Format: artifact.Format, ComponentName: artifact.ComponentName})
	}
	return &pipelinev1.PipelineStageNodeResp{Id: item.Id, NodeType: item.NodeType, PipelineId: item.PipelineId, Name: item.Name, Image: item.Image, Script: item.Script, Artifacts: artifacts, DependsOn: detail.DependsOn, SortOrder: int32(item.SortOrder), Description: item.Description, SourceTemplateStageId: item.SourceTemplateStageId, SourceTemplateStageName: item.SourceTemplateStageName, SourceTemplateStageDescription: item.SourceTemplateStageDescription, SourceTemplateStageVersion: int32(item.SourceTemplateStageVersion), LatestTemplateStageVersion: intPtrToInt32(detail.LatestTemplateStageVersion), CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
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
	return &pipelinev1.PipelineSnapshotResp{Id: item.Id, PipelineId: item.PipelineId, PipelineName: item.PipelineName, PipelineVersion: int32(item.PipelineVersion), SourcePipelineId: item.SourcePipelineId, SourceTemplateName: item.SourceTemplateName, SourceTemplateVersion: int32(item.SourceTemplateVersion), ApplicationId: item.ApplicationId, ApplicationName: item.ApplicationName, RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, VersionForkStrategy: item.VersionForkStrategy, FixedVersionId: item.FixedVersionId, FixedVersionLabel: item.FixedVersionLabel, StagesSnapshot: stages, VariablesSnapshot: variableDeclarationResponses(detail.VariablesSnapshot), CreatedAt: transportresponse.FormatTime(item.CreatedAt)}
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
			result = append(result, map[string]any{"name": item.Name, "description": item.Description, "default": transportresponse.NativeValue(item.Default), "value": transportresponse.NativeValue(item.Value), "secret": item.Secret, "source": item.Source, "editable": item.Editable})
		}
	}
	return result
}
func variableResponses(items []map[string]any) []*commonv1.VariableDeclarationResp {
	declarations, _ := variableDeclarations(items)
	result := make([]*commonv1.VariableDeclarationResp, 0, len(declarations))
	for _, item := range declarations {
		result = append(result, &commonv1.VariableDeclarationResp{Name: item.Name, Description: item.Description, Default: transportresponse.ProtoValue(item.Default), Value: transportresponse.ProtoValue(item.Value), Secret: item.Secret, Source: item.Source, Editable: item.Editable})
	}
	return result
}
func variableDeclarationResponses(items []model.VariableDeclaration) []*commonv1.VariableDeclarationResp {
	result := make([]*commonv1.VariableDeclarationResp, 0, len(items))
	for _, item := range items {
		result = append(result, &commonv1.VariableDeclarationResp{Name: item.Name, Description: item.Description, Default: transportresponse.ProtoValue(item.Default), Value: transportresponse.ProtoValue(item.Value), Secret: item.Secret, Source: item.Source, Editable: item.Editable})
	}
	return result
}
func variableDeclarations(values []map[string]any) ([]model.VariableDeclaration, error) {
	result := make([]model.VariableDeclaration, 0, len(values))
	for _, value := range values {
		result = append(result, model.VariableDeclaration{Name: stringValue(value["name"]), Description: stringValue(value["description"]), Default: value["default"], Value: value["value"], Secret: boolValue(value["secret"]), Source: stringValue(value["source"]), Editable: boolValue(value["editable"])})
	}
	return result, nil
}
func stringValue(value any) string { result, _ := value.(string); return result }
func boolValue(value any) bool     { result, _ := value.(bool); return result }
func intPtrToInt32(value *int) *int32 {
	if value == nil {
		return nil
	}
	result := int32(*value)
	return &result
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
