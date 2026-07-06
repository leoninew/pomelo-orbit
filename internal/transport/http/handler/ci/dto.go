package cihandler

import (
	"encoding/json"

	apiv1 "backend/internal/transport/http/dto/proto/orbit/api/v1"
)

type PipelineTemplateResp = apiv1.PipelineTemplateResp
type StageOrchestrationResp = apiv1.StageOrchestrationResp
type StageOrchestrationReq = apiv1.StageOrchestrationReq
type ArtifactConfigResp = apiv1.ArtifactConfigResp
type ArtifactConfigReq = apiv1.ArtifactConfigReq
type VariableDeclarationResp = apiv1.VariableDeclarationResp
type VariableDeclarationReq = apiv1.VariableDeclarationReq
type TemplateVariableResolveResp = apiv1.TemplateVariableResolveResp
type BuildStageResp = apiv1.BuildStageResp
type PipelineTemplateCreateReq = apiv1.PipelineTemplateCreateReq
type PipelineTemplateUpdateReq = apiv1.PipelineTemplateUpdateReq
type TemplateVariableResolveReq = apiv1.TemplateVariableResolveReq
type PipelineRunTriggerReq = apiv1.PipelineRunTriggerReq
type PipelineRunResp = apiv1.PipelineRunResp
type StageRunResp = apiv1.StageRunResp
type ArtifactResp = apiv1.ArtifactResp
type PipelineRunArtifactListResp = apiv1.PipelineRunArtifactListResp
type PipelineStageLogResp = apiv1.PipelineStageLogResp
type RepositoryResp = apiv1.RepositoryResp
type RepositoryCreateReq = apiv1.RepositoryCreateReq
type RepositoryUpdateReq = apiv1.RepositoryUpdateReq
type RepositoryWebhookResp = apiv1.RepositoryWebhookResp
type RepositoryWebhookListResp = apiv1.RepositoryWebhookListResp
type RepositoryWebhookReceiveResp = apiv1.RepositoryWebhookReceiveResp
type RepositoryWebhookCreateReq = apiv1.RepositoryWebhookCreateReq
type BuildStageCreateReq = apiv1.BuildStageCreateReq
type BuildStageUpdateReq = apiv1.BuildStageUpdateReq
type CredentialResp = apiv1.CredentialResp
type CredentialDetailResp = apiv1.CredentialDetailResp
type CredentialCreateReq = apiv1.CredentialCreateReq
type CredentialUpdateReq = apiv1.CredentialUpdateReq
type CredentialExportResp = apiv1.CredentialExportResp
type CredentialImportReq = apiv1.CredentialImportReq
type PipelineSnapshotResp = apiv1.PipelineSnapshotResp
type SnapshotStageResp = apiv1.SnapshotStageResp
type PipelineTemplatePaginatedResp = apiv1.PipelineTemplatePaginatedResp
type BuildStagePaginatedResp = apiv1.BuildStagePaginatedResp
type PipelineRunPaginatedResp = apiv1.PipelineRunPaginatedResp
type RepositoryPaginatedResp = apiv1.RepositoryPaginatedResp
type CredentialPaginatedResp = apiv1.CredentialPaginatedResp
type ArtifactPaginatedResp = apiv1.ArtifactPaginatedResp

type RepositoryWebhookUpdateReq struct {
	*apiv1.RepositoryWebhookUpdateReq
	BranchSet bool
}

func (r *RepositoryWebhookUpdateReq) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	decoded := &apiv1.RepositoryWebhookUpdateReq{}
	if err := json.Unmarshal(data, decoded); err != nil {
		return err
	}
	r.RepositoryWebhookUpdateReq = decoded
	_, r.BranchSet = raw["branch_filter"]
	return nil
}
