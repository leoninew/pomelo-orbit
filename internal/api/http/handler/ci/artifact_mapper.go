package cihandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func artifactResponse(item model.Artifact) pomeloorbit.ArtifactResp {
	return pomeloorbit.ArtifactResp{
		Id:             item.Id,
		PipelineRunId:  item.PipelineRunId,
		RepositoryId:   item.RepositoryId,
		RepositoryName: item.RepositoryName,
		TemplateId:     item.TemplateId,
		TemplateName:   item.TemplateName,
		StageName:      item.StageName,
		Type:           item.Type,
		Name:           item.Name,
		Path:           item.Path,
		CreatedAt:      transportresponse.FormatTime(item.CreatedAt),
	}
}
