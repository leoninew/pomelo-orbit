package pipelinehandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	pipelinedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/dto"
	pipelinev1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/pipeline"
)

func pipelineStageDetailResponse(item pipelinedto.PipelineStageDetail) pipelinev1.PipelineStageResp {
	return pipelinev1.PipelineStageResp{
		Id:                  item.Id,
		Name:                item.Name,
		Image:               item.Image,
		Script:              item.Script,
		Artifacts:           transportresponse.Ptrs(artifactConfigsResponse(item.Artifacts)),
		BuildVersionBinding: buildVersionBindingResponse(item.BuildVersionBinding),
		Description:         item.Description,
		Version:             int32(item.Version),
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
	}
}

func buildVersionBindingInput(item *pipelinev1.BuildVersionBindingReq) *pipelinedto.BuildVersionBinding {
	if item == nil {
		return nil
	}
	return &pipelinedto.BuildVersionBinding{
		ApplicationId: item.ApplicationId, ComponentName: item.ComponentName,
		ForkStrategy: item.ForkStrategy, FixedVersionId: item.FixedVersionId,
	}
}

func buildVersionBindingResponse(item *pipelinedto.BuildVersionBinding) *pipelinev1.BuildVersionBindingResp {
	if item == nil {
		return nil
	}
	return &pipelinev1.BuildVersionBindingResp{
		ApplicationId: item.ApplicationId, ApplicationName: item.ApplicationName, ComponentName: item.ComponentName,
		ForkStrategy: item.ForkStrategy, FixedVersionId: item.FixedVersionId,
	}
}

func pipelineStageDetailsResponse(items []pipelinedto.PipelineStageDetail) []pipelinev1.PipelineStageResp {
	resp := make([]pipelinev1.PipelineStageResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, pipelineStageDetailResponse(item))
	}
	return resp
}

func artifactConfigsResponse(items []pipelinedto.ArtifactConfig) []pipelinev1.ArtifactConfigResp {
	if items == nil {
		return nil
	}
	resp := make([]pipelinev1.ArtifactConfigResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, pipelinev1.ArtifactConfigResp{
			Name: item.Name, Collector: item.Collector, Reference: item.Reference, Command: item.Command, Format: item.Format,
		})
	}
	return resp
}

func serviceArtifacts(items []*pipelinev1.ArtifactConfigReq) []pipelinedto.ArtifactConfig {
	resp := make([]pipelinedto.ArtifactConfig, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, pipelinedto.ArtifactConfig{
			Name: item.Name, Collector: item.Collector, Reference: item.Reference, Command: item.Command, Format: item.Format,
		})
	}
	return resp
}
