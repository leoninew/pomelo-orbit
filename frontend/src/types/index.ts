// 重新导出所有类型，保持向后兼容
export type { PaginatedResp } from './common';
export type { LoginReq, TokenResp, UserInfo, PasswordChangeReq, LoginHistory } from './auth';
export type {
	Application,
	GitSource,
	ImageSource,
	ApplicationCreateReq,
	ApplicationUpdateReq,
	ApplicationExportResp,
	ApplicationImportReq,
	ConfigFile,
} from './cd/application';
export type { Deployment, DeploymentDetail } from './cd/deployment';
export { deploymentStatusColors } from './cd/deployment';
export type { Route, RouteCreateReq, RouteUpdateReq } from './cd/route';
export type {
	ConfigItemResp,
	SystemConfigResp,
	SystemConfigUpdateReq,
	SystemConfigResetReq,
} from './cd/settings';
export type { Credential, CredentialCreateReq, CredentialUpdateReq } from './ci/credential';
export { credentialTypeLabels } from './ci/credential';
export type {
	PipelineTemplate,
	VariableDeclaration,
	PipelineTemplateCreateReq,
	PipelineTemplateUpdateReq,
} from './ci/template';
export type { Project, ProjectCreateReq, ProjectUpdateReq } from './ci/project';
export type { PipelineRun, PipelineRunTriggerReq } from './ci/run';
export { pipelineRunStatusColors } from './ci/run';
export type { Job, JobLog, Artifact } from './ci/job';
export { jobStatusColors } from './ci/job';
