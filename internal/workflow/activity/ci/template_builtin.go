package cisvc

func isPipelineTemplateBuiltinVariable(name string) bool {
	_, ok := pipelineTemplateBuiltinVariableSpecs()[name]
	return ok
}

func pipelineTemplateBuiltinVariableSpecs() map[string]string {
	return map[string]string{
		"repository_id":    "运行时注入: 当前项目 ID",
		"repository_name":  "运行时注入: 当前项目名称",
		"repository_code":  "运行时注入: 当前项目编码",
		"repository_url":   "运行时注入: 当前仓库地址",
		"repository_ref":   "运行时注入: 当前分支",
		"template_id":      "运行时注入: 当前模板 ID",
		"template_name":    "运行时注入: 当前模板名称",
		"template_version": "运行时注入: 当前模板版本",
		"runtime_datetime": "运行时注入: 流水线启动时间 (UTC, 格式 YYYYmmdd-HHmmss)",
	}
}
