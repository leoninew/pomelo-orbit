package pipelinehandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	pipelinedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/dto"
	pipelinev1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/pipeline"
)

func pipelineTemplateResponse(detail pipelinedto.PipelineTemplateDetail) pipelinev1.PipelineTemplateResp {
	item := detail.Template
	return pipelinev1.PipelineTemplateResp{
		Id:                   item.Id,
		Name:                 item.Name,
		Description:          item.Description,
		Orchestration:        transportresponse.Ptrs(orchestrationResponse(detail.Orchestration)),
		Stages:               transportresponse.Ptrs(pipelineStageDetailsResponse(detail.Stages)),
		VariableDeclarations: transportresponse.Ptrs(variableDeclarationResponses(detail.VariableDeclarations)),
		Version:              int32(item.Version),
		CreatedAt:            transportresponse.FormatTime(item.CreatedAt),
		UpdatedAt:            transportresponse.FormatTime(item.UpdatedAt),
	}
}

func orchestrationResponse(items []pipelinedto.StageOrchestration) []pipelinev1.StageOrchestrationResp {
	resp := make([]pipelinev1.StageOrchestrationResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, pipelinev1.StageOrchestrationResp{
			StageId:      item.StageId,
			StageName:    item.StageName,
			StageVersion: int32(item.StageVersion),
			DependsOn:    item.DependsOn,
			SortOrder:    int32(item.SortOrder),
		})
	}
	return resp
}

func serviceOrchestration(items []*pipelinev1.StageOrchestrationReq) []pipelinedto.StageOrchestration {
	resp := make([]pipelinedto.StageOrchestration, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, pipelinedto.StageOrchestration{
			StageId:      item.StageId,
			StageName:    item.StageName,
			StageVersion: int(item.StageVersion),
			DependsOn:    item.DependsOn,
			SortOrder:    int(item.SortOrder),
		})
	}
	return resp
}
