package settingshandler

type ConfigItemResp struct {
	Key          string `json:"key"`
	Value        any    `json:"value"`
	Default      any    `json:"default"`
	IsOverridden bool   `json:"is_overridden"`
	Secret       bool   `json:"secret"`
	Description  string `json:"description,omitempty"`
}

type SystemConfigResp struct {
	Items []ConfigItemResp `json:"items"`
}

type SystemConfigUpdateReq struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type SystemConfigResetReq struct {
	Keys []string `json:"keys"`
}
