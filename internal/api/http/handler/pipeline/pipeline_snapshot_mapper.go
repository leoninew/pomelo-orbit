package pipelinehandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	pipelinedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/dto"
	commonv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/common"
	pipelinev1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/pipeline"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func pipelineSnapshotResponse(detail pipelinedto.PipelineSnapshotDetail) pipelinev1.PipelineSnapshotResp {
	item := detail.Snapshot
	return pipelinev1.PipelineSnapshotResp{
		Id:                item.Id,
		TemplateId:        item.TemplateId,
		Version:           int32(item.Version),
		StagesSnapshot:    transportresponse.Ptrs(snapshotStageResponses(detail.StagesSnapshot)),
		VariablesSnapshot: transportresponse.Ptrs(snapshotVariableDeclarationResponses(detail.VariablesSnapshot)),
		CreatedAt:         transportresponse.FormatTime(item.CreatedAt),
	}
}

func snapshotVariableDeclarationResponses(items []model.VariableDeclaration) []commonv1.VariableDeclarationResp {
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

func snapshotStageResponses(items []model.StageDefinition) []pipelinev1.SnapshotStageResp {
	resp := make([]pipelinev1.SnapshotStageResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, pipelinev1.SnapshotStageResp{
			Name:                item.Name,
			Id:                  item.Id,
			Image:               item.Image,
			Version:             int32(item.Version),
			DependsOn:           item.DependsOn,
			Script:              item.Script,
			Artifacts:           transportresponse.Ptrs(snapshotArtifactConfigResponses(item.Artifacts)),
			BuildVersionBinding: buildVersionBindingModelResponse(item.BuildVersionBinding),
		})
	}
	return resp
}

func buildVersionBindingModelResponse(item *model.BuildVersionBinding) *pipelinev1.BuildVersionBindingResp {
	if item == nil {
		return nil
	}
	return &pipelinev1.BuildVersionBindingResp{
		ApplicationId: item.ApplicationId, ApplicationName: item.ApplicationName, ComponentName: item.ComponentName,
		ForkStrategy: item.ForkStrategy, FixedVersionId: item.FixedVersionId,
	}
}

func snapshotArtifactConfigResponses(items []model.ArtifactConfig) []pipelinev1.ArtifactConfigResp {
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
