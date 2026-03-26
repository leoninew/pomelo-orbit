// API 响应类型定义

// 通用分页响应
export interface PaginatedResp<T> {
	items: T[]
	total: number
	page: number
	per_page: number
	pages: number
}

// 认证相关
export interface LoginReq {
	username: string
	password: string
}

export interface TokenResp {
	access_token: string
	token_type: string
}

export interface UserInfo {
	id: string
	username: string
	created_at: string
	last_login_at: string | null
}

export interface PasswordChangeReq {
	old_password: string
	new_password: string
}

export interface LoginHistory {
	id: string
	user_id: string
	username: string
	ip_address: string | null
	user_agent: string | null
	login_at: string
	success: boolean
}

// 应用相关
export interface GitSource {
	id: string
	repository_url: string
	deploy_branches: string
	auto_deploy: boolean
}

export interface ImageSource {
	id: string
	image_name: string
	registry_url: string | null
}

export interface Application {
	id: string
	name: string
	code: string
	image_pull_policy: string
	enabled: boolean
	status: string
	created_at: string
	updated_at: string
	git_source: GitSource | null
	image_source: ImageSource | null
}

export interface ApplicationCreateReq {
	name: string
	repository_url?: string
	deploy_branches?: string
	auto_deploy?: boolean
	image_pull_policy?: string
	enabled?: boolean
}

export interface ApplicationUpdateReq {
	name?: string
	repository_url?: string
	deploy_branches?: string
	auto_deploy?: boolean
	image_pull_policy?: string
	enabled?: boolean
}

export interface ApplicationExportResp {
	version: string
	name: string
	code: string
	image_pull_policy: string
	enabled: boolean
	git_source: Pick<GitSource, 'repository_url' | 'deploy_branches' | 'auto_deploy'> | null
	image_source: Pick<ImageSource, 'image_name' | 'registry_url'> | null
	config_files: { path: string; content: string }[]
}

export interface ApplicationImportReq {
	version?: string
	name: string
	code: string
	enabled?: boolean
	image_pull_policy?: string
	git_source?: { repository_url: string; deploy_branches?: string; auto_deploy?: boolean } | null
	image_source?: { image_name: string; registry_url?: string | null } | null
	config_files?: { path: string; content?: string }[]
}

export interface ConfigFile {
	id: string
	path: string
	created_at: string
}

// 部署记录相关
export interface Deployment {
	id: string
	application_id: string
	application_name: string | null
	operation_type: string
	trigger_type: string
	trigger_ref: string | null
	env_file: string | null
	status: string
	started_at: string
	finished_at: string | null
	duration_ms: number | null
	error_message: string | null
}

export interface DeploymentDetail extends Deployment {
	log_text: string | null
}

// 回调事件相关
export interface WebhookEvent {
	id: string
	source: string
	event_type: string
	repository_name: string | null
	repository_url: string | null
	branch: string | null
	sender: string | null
	signature_valid: boolean | null
	status: string
	matched_application_id: string | null
	triggered_deployment_id: string | null
	error_message: string | null
	received_at: string
	processed_at: string | null
}

export interface WebhookEventDetail extends WebhookEvent {
	payload: string | null
}

// 路由相关
export interface Route {
	id: string
	name: string
	domain: string
	path_prefix: string
	target_url: string
	enabled: boolean
	created_at: string
	updated_at: string
}

export interface RouteCreateReq {
	name: string
	domain: string
	path_prefix?: string
	target_url: string
	enabled?: boolean
}

export interface RouteUpdateReq {
	name?: string
	domain?: string
	path_prefix?: string
	target_url?: string
	enabled?: boolean
}

// 系统配置相关
export interface ConfigItemResp {
	key: string
	value: unknown
	default: unknown
	is_overridden: boolean
}

export interface SystemConfigResp {
	items: ConfigItemResp[]
}

export interface SystemConfigUpdateReq {
	key: string
	value: string | boolean | number
}

export interface SystemConfigResetReq {
	keys: string[]
}

// 状态颜色映射
export const deploymentStatusColors: Record<string, string> = {
	queued: 'default',
	running: 'processing',
	success: 'success',
	failed: 'error',
};

export const eventStatusColors: Record<string, string> = {
	received: 'default',
	matched: 'processing',
	ignored: 'warning',
	error: 'error',
	processed: 'success',
};
