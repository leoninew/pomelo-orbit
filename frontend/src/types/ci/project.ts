// Project
export interface Project {
	id: string
	name: string
	repository_url: string
	pipeline_snapshot_id: string
	git_credential_id?: string | null
	variable_overrides: Record<string, string>
	default_branch: string
	created_at: string
	updated_at: string
}

export interface ProjectCreateReq {
	name: string
	repository_url: string
	pipeline_snapshot_id: string
	git_credential_id?: string | null
	variable_overrides?: Record<string, string>
	default_branch?: string
}

export interface ProjectUpdateReq {
	name?: string
	repository_url?: string
	pipeline_snapshot_id?: string
	git_credential_id?: string | null
	variable_overrides?: Record<string, string>
	default_branch?: string
}
