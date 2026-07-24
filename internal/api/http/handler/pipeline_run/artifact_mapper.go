package pipelinerunhandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	pipelinerunv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/pipeline_run"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func artifactResponse(item model.Artifact) pipelinerunv1.ArtifactResp {
	return pipelinerunv1.ArtifactResp{
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
