package pipelinerunhandler

import (
	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"
	pipelinerundto "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/dto"
	"github.com/leoninew/pomelo-orbit/internal/application/variableview"
	commonv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/common"
	pipelinerunv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/pipeline_run"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func pipelineRunResponse(detail pipelinerundto.PipelineRunDetail) *pipelinerunv1.PipelineRunResp {
	item := detail.Run
	response := &pipelinerunv1.PipelineRunResp{Id: item.Id, ProjectId: item.ProjectId, RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, SnapshotId: item.SnapshotId, PipelineId: item.PipelineId, PipelineName: item.PipelineName, PipelineVersion: int32(item.PipelineVersion), Trigger: item.Trigger, RepositoryRef: item.RepositoryRef, Variables: variableResponses(detail.Variables), Status: item.Status, RetryOf: item.RetryOf, StartedAt: transport.FormatOptionalTime(item.StartedAt), FinishedAt: transport.FormatOptionalTime(item.FinishedAt), ErrorMessage: item.ErrorMessage, CreatedAt: transport.FormatTime(item.CreatedAt), PipelineStageRuns: pipelineStageRunsResponse(detail.PipelineStageRuns)}
	if detail.VersionBinding != nil {
		response.VersionBinding = &pipelinerunv1.PipelineRunVersionBindingResp{ApplicationId: detail.VersionBinding.ApplicationId, ApplicationName: detail.VersionBinding.ApplicationName, SourceVersionId: detail.VersionBinding.SourceVersionId, SourceVersionLabel: detail.VersionBinding.SourceVersionLabel, GeneratedVersionId: detail.VersionBinding.GeneratedVersionId, GeneratedVersionLabel: detail.VersionBinding.GeneratedVersionLabel}
	}
	return response
}
func variableResponses(items []variableview.View) []*commonv1.VariableResp {
	response := make([]*commonv1.VariableResp, 0, len(items))
	for _, item := range items {
		references := make([]*commonv1.VariableReferenceResp, 0, len(item.References))
		for _, reference := range item.References {
			references = append(references, &commonv1.VariableReferenceResp{StageId: reference.StageId, StageName: reference.StageName, Field: reference.Field, ArtifactName: reference.ArtifactName, ArtifactIndex: transport.OptionalInt32(reference.ArtifactIndex), Default: transport.ProtoValue(reference.Default), HasDefault: reference.HasDefault})
		}
		var binding *commonv1.VariableStageBindingResp
		if item.StageBinding != nil {
			binding = &commonv1.VariableStageBindingResp{StageId: item.StageBinding.StageId, StageName: item.StageBinding.StageName}
		}
		response = append(response, &commonv1.VariableResp{Name: item.Name, Kind: item.Kind, Scope: item.Scope, StageBinding: binding, References: references, Configuration: variableConfigurationResponse(item.Configuration), GlobalConfiguration: variableConfigurationResponse(item.GlobalConfiguration), StageOverride: variableConfigurationResponse(item.StageOverride), ValueSource: item.ValueSource, Editable: item.Editable})
	}
	return response
}

func variableConfigurationResponse(item *variableview.Configuration) *commonv1.VariableConfigurationResp {
	if item == nil {
		return nil
	}
	return &commonv1.VariableConfigurationResp{Description: item.Description, Default: transport.ProtoValue(item.Default), Value: transport.ProtoValue(item.Value), Secret: item.Secret, Editable: item.Editable}
}
func pipelineStageRunsResponse(items []model.PipelineStageRun) []*pipelinerunv1.PipelineStageRunResp {
	response := make([]*pipelinerunv1.PipelineStageRunResp, 0, len(items))
	for _, item := range items {
		response = append(response, pipelineStageRunResponse(item))
	}
	return response
}
func pipelineStageRunResponse(item model.PipelineStageRun) *pipelinerunv1.PipelineStageRunResp {
	return &pipelinerunv1.PipelineStageRunResp{Id: item.Id, PipelineRunId: item.PipelineRunId, StageId: item.StageId, StageName: item.StageName, Status: item.Status, StartedAt: transport.FormatOptionalTime(item.StartedAt), FinishedAt: transport.FormatOptionalTime(item.FinishedAt), ExitCode: transport.OptionalInt32(item.ExitCode), ErrorMessage: item.ErrorMessage}
}

func artifactResponse(item model.Artifact) pipelinerunv1.ArtifactResp {
	return pipelinerunv1.ArtifactResp{Id: item.Id, PipelineRunId: item.PipelineRunId, RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, PipelineId: item.PipelineId, PipelineName: item.PipelineName, PipelineStageId: item.PipelineStageId, StageName: item.StageName, Collector: item.Collector, Name: item.Name, Location: item.Location, Value: item.Value, ValueFormat: item.ValueFormat, CreatedAt: transport.FormatTime(item.CreatedAt), ImageRef: item.ImageRef, LocalImageSha256: item.LocalImageSha256, SourceCommitSha: item.SourceCommitSha, ApplicationId: item.ApplicationId, ApplicationName: item.ApplicationName, SourceVersionId: item.SourceVersionId, SourceVersionLabel: item.SourceVersionLabel, GeneratedVersionId: item.GeneratedVersionId, GeneratedVersionLabel: item.GeneratedVersionLabel, VersionComponentId: item.VersionComponentId, VersionComponentName: item.VersionComponentName}
}
