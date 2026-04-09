export interface Repository {
	id: string
	name: string
	code: string
	repository_url: string
	git_credential_id?: string | null
	git_credential_name?: string | null
	variable_overrides: Record<string, string>
	default_branch: string
	created_at: string
	updated_at: string
}

export interface RepositoryCreateReq {
	name: string
	code: string
	repository_url: string
	git_credential_id?: string | null
	variable_overrides?: Record<string, string>
	default_branch?: string
}

export interface RepositoryUpdateReq {
	name?: string
	repository_url?: string
	git_credential_id?: string | null
	variable_overrides?: Record<string, string>
	default_branch?: string
}
