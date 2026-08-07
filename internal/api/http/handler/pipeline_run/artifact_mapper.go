package pipelinerunhandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	pipelinerunv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/pipeline_run"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func artifactResponse(item model.Artifact) pipelinerunv1.ArtifactResp {
	return pipelinerunv1.ArtifactResp{Id: item.Id, PipelineRunId: item.PipelineRunId, RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, PipelineId: item.PipelineId, PipelineName: item.PipelineName, PipelineStageId: item.PipelineStageId, StageName: item.StageName, Collector: item.Collector, Name: item.Name, Location: item.Location, Value: item.Value, ValueFormat: item.ValueFormat, CreatedAt: transportresponse.FormatTime(item.CreatedAt), ImageRef: item.ImageRef, LocalImageSha256: item.LocalImageSha256, SourceCommitSha: item.SourceCommitSha, ApplicationId: item.ApplicationId, ApplicationName: item.ApplicationName, SourceVersionId: item.SourceVersionId, SourceVersionLabel: item.SourceVersionLabel, GeneratedVersionId: item.GeneratedVersionId, GeneratedVersionLabel: item.GeneratedVersionLabel, VersionComponentId: item.VersionComponentId, VersionComponentName: item.VersionComponentName}
}
