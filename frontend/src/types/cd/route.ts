// 路由相关
export interface Route {
	id: string
	name: string
	domain: string
	path_prefix: string
	target_url: string
	enabled: boolean
	https_enabled: boolean
	cert_type: string
	created_at: string
	updated_at: string
}

export interface RouteCreateReq {
	name: string
	domain: string
	path_prefix?: string
	target_url: string
	enabled?: boolean
}

export interface RouteUpdateReq {
	name?: string
	domain?: string
	path_prefix?: string
	target_url?: string
	enabled?: boolean
}
