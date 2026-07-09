package dto

type ConfigItem struct {
	Key          string `json:"key"`
	Value        any    `json:"value"`
	Default      any    `json:"default"`
	IsOverridden bool   `json:"is_overridden"`
	Secret       bool   `json:"secret"`
	Description  string `json:"description,omitempty"`
}

type SystemConfig struct {
	Items []ConfigItem `json:"items"`
}

type Definition struct {
	Key         string
	Default     any
	Description string
	Secret      bool
}
