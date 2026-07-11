package cihandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cidto "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
)

func pipelineTemplateResponse(detail cidto.PipelineTemplateDetail) pomeloorbit.PipelineTemplateResp {
	item := detail.Template
	return pomeloorbit.PipelineTemplateResp{
		Id:                   item.Id,
		Name:                 item.Name,
		Description:          item.Description,
		Orchestration:        transportresponse.Ptrs(orchestrationResponse(detail.Orchestration)),
		Stages:               transportresponse.Ptrs(buildStageDetailsResponse(detail.Stages)),
		VariableDeclarations: transportresponse.Ptrs(variableDeclarationResponses(detail.VariableDeclarations)),
		Version:              int32(item.Version),
		CreatedAt:            transportresponse.FormatTime(item.CreatedAt),
		UpdatedAt:            transportresponse.FormatTime(item.UpdatedAt),
	}
}

func orchestrationResponse(items []cidto.StageOrchestration) []pomeloorbit.StageOrchestrationResp {
	resp := make([]pomeloorbit.StageOrchestrationResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, pomeloorbit.StageOrchestrationResp{
			StageId:      item.StageId,
			StageName:    item.StageName,
			StageVersion: int32(item.StageVersion),
			DependsOn:    item.DependsOn,
			SortOrder:    int32(item.SortOrder),
		})
	}
	return resp
}

func serviceOrchestration(items []*pomeloorbit.StageOrchestrationReq) []cidto.StageOrchestration {
	resp := make([]cidto.StageOrchestration, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, cidto.StageOrchestration{
			StageId:      item.StageId,
			StageName:    item.StageName,
			StageVersion: int(item.StageVersion),
			DependsOn:    item.DependsOn,
			SortOrder:    int(item.SortOrder),
		})
	}
	return resp
}
