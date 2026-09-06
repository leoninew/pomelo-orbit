package pipelinerunhandler

import (
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	pipelinerundto "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/dto"
	commonv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/common"
	pipelinerunv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/pipeline_run"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func pipelineRunResponse(detail pipelinerundto.PipelineRunDetail) *pipelinerunv1.PipelineRunResp {
	item := detail.Run
	response := &pipelinerunv1.PipelineRunResp{Id: item.Id, ProjectId: item.ProjectId, RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, SnapshotId: item.SnapshotId, PipelineId: item.PipelineId, PipelineName: item.PipelineName, PipelineVersion: int32(item.PipelineVersion), Trigger: item.Trigger, RepositoryRef: item.RepositoryRef, VariablesSnapshot: pipelineRunVariableDeclarationResponses(detail.VariablesSnapshot), Status: item.Status, RetryOf: item.RetryOf, StartedAt: transportresponse.FormatOptionalTime(item.StartedAt), FinishedAt: transportresponse.FormatOptionalTime(item.FinishedAt), ErrorMessage: item.ErrorMessage, CreatedAt: transportresponse.FormatTime(item.CreatedAt), PipelineStageRuns: pipelineStageRunsResponse(detail.PipelineStageRuns)}
	if detail.VersionBinding != nil {
		response.VersionBinding = &pipelinerunv1.PipelineRunVersionBindingResp{ApplicationId: detail.VersionBinding.ApplicationId, ApplicationName: detail.VersionBinding.ApplicationName, SourceVersionId: detail.VersionBinding.SourceVersionId, SourceVersionLabel: detail.VersionBinding.SourceVersionLabel, GeneratedVersionId: detail.VersionBinding.GeneratedVersionId, GeneratedVersionLabel: detail.VersionBinding.GeneratedVersionLabel}
	}
	return response
}
func pipelineRunVariableDeclarationResponses(items []model.VariableDeclaration) []*commonv1.VariableDeclarationResp {
	response := make([]*commonv1.VariableDeclarationResp, 0, len(items))
	for _, item := range items {
		response = append(response, &commonv1.VariableDeclarationResp{Name: item.Name, Description: item.Description, Default: transportresponse.ProtoValue(item.Default), Value: transportresponse.ProtoValue(item.Value), Secret: item.Secret, Source: item.Source, Editable: item.Editable, StageDefaults: pipelineRunStageVariableDefaultResponses(item.StageDefaults), StageId: item.StageId, StageName: item.StageName})
	}
	return response
}

func pipelineRunStageVariableDefaultResponses(items []model.StageVariableDefault) []*commonv1.StageVariableDefaultResp {
	response := make([]*commonv1.StageVariableDefaultResp, 0, len(items))
	for _, item := range items {
		response = append(response, &commonv1.StageVariableDefaultResp{StageId: item.StageId, StageName: item.StageName, Default: transportresponse.ProtoValue(item.Default)})
	}
	return response
}
func pipelineStageRunsResponse(items []model.PipelineStageRun) []*pipelinerunv1.PipelineStageRunResp {
	response := make([]*pipelinerunv1.PipelineStageRunResp, 0, len(items))
	for _, item := range items {
		response = append(response, pipelineStageRunResponse(item))
	}
	return response
}
func pipelineStageRunResponse(item model.PipelineStageRun) *pipelinerunv1.PipelineStageRunResp {
	return &pipelinerunv1.PipelineStageRunResp{Id: item.Id, PipelineRunId: item.PipelineRunId, StageId: item.StageId, StageName: item.StageName, Status: item.Status, StartedAt: transportresponse.FormatOptionalTime(item.StartedAt), FinishedAt: transportresponse.FormatOptionalTime(item.FinishedAt), ExitCode: transportresponse.OptionalInt32(item.ExitCode), ErrorMessage: item.ErrorMessage}
}
