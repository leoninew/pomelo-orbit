package dto

type ConfigItem struct {
	Key            string `json:"key"`
	Value          any    `json:"value"`
	Default        any    `json:"default"`
	IsOverridden   bool   `json:"is_overridden"`
	Secret         bool   `json:"secret"`
	Description    string `json:"description,omitempty"`
	Type           string `json:"type"`
	OverrideValue  any    `json:"override_value"`
	NextValue      any    `json:"next_value"`
	NextValueKnown bool   `json:"next_value_known"`
	PendingRestart bool   `json:"pending_restart"`
	ValueSource    string `json:"value_source"`
	DefaultSource  string `json:"default_source"`
	NextSource     string `json:"next_source"`
}

type SystemConfig struct {
	Items           []ConfigItem `json:"items"`
	Revision        string       `json:"revision"`
	PendingRestart  bool         `json:"pending_restart"`
	NextConfigError string       `json:"next_config_error"`
}

type Update struct {
	Key   string
	Value any
}
