export interface Project {
	id: string
	name: string
	code: string
	owner_user_id: string
	created_at: string
	updated_at: string
}

export interface ProjectCreateReq {
	name: string
	code: string
}

export interface ProjectUpdateReq {
	name: string
	code: string
}
