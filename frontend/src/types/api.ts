// 向后兼容：从新的模块化文件重新导出所有类型

export type { LoginHistory, LoginReq, PasswordChangeReq, TokenResp, UserInfo } from './auth';
export type {
	Application,
	ApplicationCreateReq,
	ApplicationExportResp,
	ApplicationImportReq,
	ApplicationUpdateReq,
	ConfigFile,
} from './cd/application';
export type { Deployment, DeploymentDetail } from './cd/deployment';
export type { Route, RouteCreateReq, RouteUpdateReq } from './cd/route';
export type {
	ConfigItemResp,
	SystemConfigResetReq,
	SystemConfigResp,
	SystemConfigUpdateReq,
} from './cd/settings';
export type {
	Credential,
	CredentialCreateReq,
	CredentialExportResp,
	CredentialImportReq,
	CredentialUpdateReq,
} from './ci/credential';
export { credentialTypeLabels } from './ci/credential';
export type {
	Repository,
	RepositoryCreateReq,
	RepositoryListItem,
	RepositoryUpdateReq,
} from './ci/repository';
export type { PipelineRun, PipelineRunTriggerReq } from './ci/run';
export type { PipelineSnapshot } from './ci/snapshot';
export type { Artifact, StageRun } from './ci/stage_run';
export type {
	ArtifactConfig,
	OrchestrationUpdateReq,
	PipelineStage,
	PipelineStageCreateReq,
	PipelineStageUpdateReq,
	PipelineTemplate,
	PipelineTemplateCreateReq,
	PipelineTemplateUpdateReq,
	StageOrchestration,
	VariableDeclaration,
} from './ci/template';
export type {
	RepositoryWebhook,
	RepositoryWebhookCreateReq,
	RepositoryWebhookUpdateReq,
} from './ci/webhook';
export type { PaginatedResp } from './common';
