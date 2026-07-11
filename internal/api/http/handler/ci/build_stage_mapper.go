package cihandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cidto "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
)

func buildStageDetailResponse(item cidto.BuildStageDetail) pomeloorbit.BuildStageResp {
	return pomeloorbit.BuildStageResp{
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

func buildStageDetailsResponse(items []cidto.BuildStageDetail) []pomeloorbit.BuildStageResp {
	resp := make([]pomeloorbit.BuildStageResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, buildStageDetailResponse(item))
	}
	return resp
}

func artifactConfigsResponse(items []cidto.ArtifactConfig) []pomeloorbit.ArtifactConfigResp {
	if items == nil {
		return nil
	}
	resp := make([]pomeloorbit.ArtifactConfigResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, pomeloorbit.ArtifactConfigResp{Type: item.Type, Path: item.Path, Name: item.Name})
	}
	return resp
}

func serviceArtifacts(items []*pomeloorbit.ArtifactConfigReq) []cidto.ArtifactConfig {
	resp := make([]cidto.ArtifactConfig, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, cidto.ArtifactConfig{Type: item.Type, Path: item.Path, Name: item.Name})
	}
	return resp
}
