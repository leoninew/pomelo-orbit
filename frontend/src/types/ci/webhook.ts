export interface RepositoryWebhook {
	id: string
	repository_id: string
	name: string
	template_id: string
	branch_filter: string | null
	enabled: boolean
	created_at: string
	updated_at: string
}

export interface ProjectWebhookCreateReq {
	name: string
	template_id: string
	secret: string
	branch_filter?: string | null
}

export interface ProjectWebhookUpdateReq {
	name?: string
	template_id?: string
	branch_filter?: string | null
	secret?: string
	enabled?: boolean
}
