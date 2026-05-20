import type { AuthSource } from './auth'

export interface Project {
	id: string
	name: string
	code: string
	is_active: boolean
	created_at: string
	updated_at: string
}

export interface ProjectMember {
	id: string
	username: string
	email: string | null
	status: string
	auth_source: AuthSource
	last_login_at: string | null
}

export interface ProjectCreateReq {
	name: string
	code: string
}

export interface ProjectUpdateReq {
	name: string
	code: string
}

export interface ProjectMemberReq {
	user_id: string
}
