import type { UserInfo } from './auth';

export interface UserResp extends UserInfo {
	is_active: boolean
	updated_at: string
}

export interface UserCreateReq {
	username: string
	password: string
	email?: string | null
}

export interface UserUpdateReq {
	password: string
}
