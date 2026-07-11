package cihandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cidto "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func pipelineRunResponse(detail cidto.PipelineRunDetail) pomeloorbit.PipelineRunResp {
	item := detail.Run
	return pomeloorbit.PipelineRunResp{
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
		StageRuns:         transportresponse.Ptrs(stageRunsResponse(detail.StageRuns)),
	}
}

func pipelineRunVariableDeclarationResponses(items []model.VariableDeclaration) []pomeloorbit.VariableDeclarationResp {
	resp := make([]pomeloorbit.VariableDeclarationResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, pomeloorbit.VariableDeclarationResp{
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

func stageRunsResponse(items []model.StageRun) []pomeloorbit.StageRunResp {
	resp := make([]pomeloorbit.StageRunResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, stageRunResponse(item))
	}
	return resp
}

func stageRunResponse(item model.StageRun) pomeloorbit.StageRunResp {
	return pomeloorbit.StageRunResp{
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
