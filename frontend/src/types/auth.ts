// 认证相关
export interface CsrfTokenResp {
	token: string
}

export interface CaptchaResp {
	token: string
	image: string
}

export interface LoginReq {
	username: string
	password: string
	csrf_token: string
	captcha_token: string
	captcha_answer: string
}

export interface TokenResp {
	access_token: string
	token_type: string
}

export interface UserInfo {
	id: string
	username: string
	email: string | null
	auth_source: string
	created_at: string
	last_login_at: string | null
}

export interface PasswordChangeReq {
	old_password: string
	new_password: string
}

export interface LoginHistory {
	id: string
	user_id: string
	username: string
	ip_address: string | null
	user_agent: string | null
	login_at: string
	success: boolean
}

export interface GoogleCallbackReq {
	code: string
}
