export interface PermissionResp {
	id: string
	code: string
	name: string
	description: string | null
}

export interface RoleResp {
	id: string
	code: string
	name: string
	description: string | null
	created_at: string
	updated_at: string
	permission_codes: string[]
}

export interface RoleCreateReq {
	code: string
	name: string
	description: string | null
	permission_codes: string[]
}

export interface RoleUpdateReq {
	code: string
	name: string
	description?: string | null
	permission_codes?: string[]
}
