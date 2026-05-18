export interface RoleResp {
	id: string
	code: string
	name: string
	description: string | null
	is_active: boolean
	created_at: string
	updated_at: string
}

export interface RoleCreateReq {
	code: string
	name: string
	description: string | null
}

export interface RoleUpdateReq {
	code: string
	name: string
	description: string | null
}
