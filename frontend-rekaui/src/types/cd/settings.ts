// 系统配置相关
export interface ConfigItemResp {
	key: string
	value: unknown
	default: unknown
	is_overridden: boolean
}

export interface SystemConfigResp {
	items: ConfigItemResp[]
}

export interface SystemConfigUpdateReq {
	key: string
	value: string | boolean | number
}

export interface SystemConfigResetReq {
	keys: string[]
}
