package pipelinehandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	pipelinedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/dto"
	pipelinev1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/pipeline"
)

func pipelineStageDetailResponse(item pipelinedto.PipelineStageDetail) pipelinev1.PipelineStageResp {
	return pipelinev1.PipelineStageResp{
		Id:          item.Id,
		Name:        item.Name,
		Image:       item.Image,
		Script:      item.Script,
		Artifacts:   transportresponse.Ptrs(artifactConfigsResponse(item.Artifacts)),
		Description: item.Description,
		Version:     int32(item.Version),
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
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
		resp = append(resp, pipelinev1.ArtifactConfigResp{Type: item.Type, Path: item.Path, Name: item.Name})
	}
	return resp
}

func serviceArtifacts(items []*pipelinev1.ArtifactConfigReq) []pipelinedto.ArtifactConfig {
	resp := make([]pipelinedto.ArtifactConfig, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, pipelinedto.ArtifactConfig{Type: item.Type, Path: item.Path, Name: item.Name})
	}
	return resp
}
