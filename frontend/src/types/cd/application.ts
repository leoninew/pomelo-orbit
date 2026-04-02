// 应用相关
export interface ImageSource {
	id: string
	image_name: string
	registry_url: string | null
}

export interface Application {
	id: string
	name: string
	code: string
	image_pull_policy: string
	status: string
	created_at: string
	updated_at: string
	image_source: ImageSource | null
}

export interface ApplicationCreateReq {
	name: string
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
	image_source: Pick<ImageSource, 'image_name' | 'registry_url'> | null
	config_files: { path: string; content: string }[]
}

export interface ApplicationImportReq {
	version?: string
	name: string
	code: string
	image_pull_policy?: string
	image_source?: { image_name: string; registry_url?: string | null } | null
	config_files?: { path: string; content?: string }[]
}

export interface ConfigFile {
	id: string
	path: string
	created_at: string
}
