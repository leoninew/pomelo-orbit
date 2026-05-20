import type { UserInfo } from './auth';

export type UserStatus = 'enabled' | 'disabled';

export interface UserRoleResp {
	id: string
	code: string
	name: string
	permission_codes: string[]
}

export interface UserListResp {
	id: string
	username: string
	email: string | null
	auth_source: UserInfo['auth_source']
	created_at: string
	last_login_at: string | null
	status: UserStatus
	updated_at: string
	role_items: UserRoleResp[]
}

export interface UserResp extends UserInfo {
	status: UserStatus
	updated_at: string
	role_items: UserRoleResp[]
}

export interface UserCreateReq {
	username: string
	password: string
	email?: string | null
}

export interface UserUpdateReq {
	username?: string
	password?: string | null
	status?: UserStatus
}

export interface UserRoleUpdateReq {
	role_ids: string[]
}
