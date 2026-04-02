// Project
export interface Project {
	id: string
	name: string
	repository_url: string
	pipeline_template_id: string
	git_credential_id?: string
	variable_overrides: Record<string, string>
	webhook_secret?: string
	branch_filter?: string
	default_branch: string
	created_at: string
	updated_at: string
}

export interface ProjectCreateReq {
	name: string
	repository_url: string
	pipeline_template_id: string
	git_credential_id?: string
	variable_overrides?: Record<string, string>
	branch_filter?: string
	default_branch?: string
}

export interface ProjectUpdateReq {
	name?: string
	repository_url?: string
	pipeline_template_id?: string
	git_credential_id?: string
	variable_overrides?: Record<string, string>
	branch_filter?: string
	default_branch?: string
}
