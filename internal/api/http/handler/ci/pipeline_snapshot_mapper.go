package cihandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cidto "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func pipelineSnapshotResponse(detail cidto.PipelineSnapshotDetail) pomeloorbit.PipelineSnapshotResp {
	item := detail.Snapshot
	return pomeloorbit.PipelineSnapshotResp{
		Id:                item.Id,
		TemplateId:        item.TemplateId,
		Version:           int32(item.Version),
		StagesSnapshot:    transportresponse.Ptrs(snapshotStageResponses(detail.StagesSnapshot)),
		VariablesSnapshot: transportresponse.Ptrs(pipelineRunVariableDeclarationResponses(detail.VariablesSnapshot)),
		CreatedAt:         transportresponse.FormatTime(item.CreatedAt),
	}
}

func snapshotStageResponses(items []model.StageDefinition) []pomeloorbit.SnapshotStageResp {
	resp := make([]pomeloorbit.SnapshotStageResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, pomeloorbit.SnapshotStageResp{
			Name:      item.Name,
			Id:        item.Id,
			Image:     item.Image,
			Version:   int32(item.Version),
			DependsOn: item.DependsOn,
			Script:    item.Script,
			Artifacts: transportresponse.Ptrs(snapshotArtifactConfigResponses(item.Artifacts)),
		})
	}
	return resp
}

func snapshotArtifactConfigResponses(items []model.ArtifactConfig) []pomeloorbit.ArtifactConfigResp {
	if items == nil {
		return nil
	}
	resp := make([]pomeloorbit.ArtifactConfigResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, pomeloorbit.ArtifactConfigResp{Type: item.Type, Path: item.Path, Name: item.Name})
	}
	return resp
}
