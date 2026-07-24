package pipelinerunhandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	pipelinerundto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/dto"
	commonv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/common"
	pipelinerunv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/pipeline_run"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func pipelineRunResponse(detail pipelinerundto.PipelineRunDetail) pipelinerunv1.PipelineRunResp {
	item := detail.Run
	return pipelinerunv1.PipelineRunResp{
		Id:                item.Id,
		ProjectId:         item.ProjectId,
		RepositoryId:      item.RepositoryId,
		RepositoryName:    item.RepositoryName,
		SnapshotId:        item.SnapshotId,
		TemplateId:        item.TemplateId,
		TemplateName:      item.TemplateName,
		TemplateVersion:   int32(item.TemplateVersion),
		Trigger:           item.Trigger,
		TriggerRef:        item.TriggerRef,
		VariablesSnapshot: transportresponse.Ptrs(pipelineRunVariableDeclarationResponses(detail.VariablesSnapshot)),
		Status:            item.Status,
		RetryOf:           item.RetryOf,
		StartedAt:         transportresponse.FormatOptionalTime(item.StartedAt),
		FinishedAt:        transportresponse.FormatOptionalTime(item.FinishedAt),
		ErrorMessage:      item.ErrorMessage,
		CreatedAt:         transportresponse.FormatTime(item.CreatedAt),
		PipelineStageRuns: transportresponse.Ptrs(pipelineStageRunsResponse(detail.PipelineStageRuns)),
	}
}

func pipelineRunVariableDeclarationResponses(items []model.VariableDeclaration) []commonv1.VariableDeclarationResp {
	resp := make([]commonv1.VariableDeclarationResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, commonv1.VariableDeclarationResp{
			Name:        item.Name,
			Description: item.Description,
			Default:     transportresponse.ProtoValue(item.Default),
			Value:       transportresponse.ProtoValue(item.Value),
			Secret:      item.Secret,
			Source:      item.Source,
			Editable:    item.Editable,
		})
	}
	return resp
}

func pipelineStageRunsResponse(items []model.PipelineStageRun) []pipelinerunv1.PipelineStageRunResp {
	resp := make([]pipelinerunv1.PipelineStageRunResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, pipelineStageRunResponse(item))
	}
	return resp
}

func pipelineStageRunResponse(item model.PipelineStageRun) pipelinerunv1.PipelineStageRunResp {
	return pipelinerunv1.PipelineStageRunResp{
		Id:            item.Id,
		PipelineRunId: item.PipelineRunId,
		StageId:       item.StageId,
		StageName:     item.StageName,
		Status:        item.Status,
		StartedAt:     transportresponse.FormatOptionalTime(item.StartedAt),
		FinishedAt:    transportresponse.FormatOptionalTime(item.FinishedAt),
		ExitCode:      transportresponse.OptionalInt32(item.ExitCode),
		ErrorMessage:  item.ErrorMessage,
	}
}
