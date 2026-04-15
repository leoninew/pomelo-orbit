// 应用相关
export interface Application {
	id: string
	name: string
	code: string
	image_pull_policy: string
	route_managed: boolean
	status: string
	created_at: string
	updated_at: string
}

export interface ApplicationCreateReq {
	name: string
	code: string
	image_pull_policy?: string
	route_managed: boolean
}

export interface ApplicationUpdateReq {
	name?: string
	image_pull_policy?: string
	route_managed?: boolean
}

export interface ApplicationExportResp {
	version: string
	name: string
	code: string
	image_pull_policy: string
	route_managed: boolean
	config_files: { path: string; content: string }[]
	routes: { service_name: string; domain: string; port: number }[]
}

export interface ApplicationImportReq {
	version?: string
	name: string
	code: string
	image_pull_policy?: string
	route_managed?: boolean
	config_files?: { path: string; content?: string }[]
	routes?: { service_name: string; domain: string; port: number }[]
}

export interface ConfigFile {
	id: string
	path: string
	created_at: string
}

export interface ApplicationRoute {
	id: string
	service_name: string
	domain: string
	port: number
	created_at: string
	updated_at: string
}

export interface ComposeServiceResp {
	service_name: string
	default_domain: string
	default_port: number
}
