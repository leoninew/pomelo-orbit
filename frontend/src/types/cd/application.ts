// 应用相关
export interface Application {
	id: string
	name: string
	code: string
	image_pull_policy: string
	status: string
	created_at: string
	updated_at: string
}

export interface ApplicationCreateReq {
	name: string
	code: string
	image_pull_policy?: string
}

export interface ApplicationUpdateReq {
	name?: string
	image_pull_policy?: string
}

export interface ApplicationExportResp {
	version: string
	name: string
	code: string
	image_pull_policy: string
	config_files: { path: string; content: string }[]
}

export interface ApplicationImportReq {
	version?: string
	name: string
	code: string
	image_pull_policy?: string
	config_files?: { path: string; content?: string }[]
}

export interface ConfigFile {
	id: string
	path: string
	created_at: string
}
