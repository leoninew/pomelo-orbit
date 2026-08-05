package pipelinevariable

import (
	"encoding/json"
	"maps"
	"regexp"
	"sort"
	"strings"
	"time"

	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

var templateVariablePattern = regexp.MustCompile(`\{\{\s*([A-Za-z][A-Za-z0-9_]*)(?:\s*\|\s*default\s*:\s*['\"]([^'\"]*)['\"])?\s*\}\}`)

func CompleteSnapshotVariableDeclarations(snapshot model.PipelineSnapshot, template model.PipelineTemplate) ([]model.VariableDeclaration, error) {
	variables, err := VariableDeclarations(snapshot.VariablesSnapshot)
	if err != nil {
		return nil, err
	}
	if len(variables) > 0 {
		return variables, nil
	}
	stages, err := PipelineSnapshotStages(snapshot.StagesSnapshot)
	if err != nil {
		return nil, err
	}
	custom, err := PipelineTemplateVariables(template.VariableDeclarations)
	if err != nil {
		return nil, err
	}
	resolved := ResolveTemplateVariablesFromStageDefinitions(stages, custom)
	return VariableDeclarationsFromMaps(resolved)
}

func BuildRuntimeVariables(repo model.Repository, template model.PipelineTemplate, triggerRef string, runtimeOverrides map[string]string, declarations []model.VariableDeclaration) (map[string]any, error) {
	variables := map[string]any{}
	maps.Copy(variables, RepositoryBuiltinVariables(repo, triggerRef))
	maps.Copy(variables, TemplateBuiltinVariables(template))
	for name, value := range runtimeOverrides {
		name = strings.TrimSpace(name)
		if name == "" || IsPipelineTemplateBuiltinVariable(name) {
			continue
		}
		variables[name] = value
	}
	repoVariables, err := VariableDeclarations(repo.VariableOverrides)
	if err != nil {
		return nil, err
	}
	for _, declaration := range repoVariables {
		if _, exists := variables[declaration.Name]; !exists && declaration.Value != nil && !IsPipelineTemplateBuiltinVariable(declaration.Name) {
			variables[declaration.Name] = declaration.Value
		}
	}
	templateVariables, err := VariableDeclarations(template.VariableDeclarations)
	if err != nil {
		return nil, err
	}
	for _, declaration := range templateVariables {
		if _, exists := variables[declaration.Name]; !exists && !IsPipelineTemplateBuiltinVariable(declaration.Name) {
			if value, ok := EffectiveVariableValue(declaration); ok {
				variables[declaration.Name] = value
			}
		}
	}
	for _, declaration := range declarations {
		if _, exists := variables[declaration.Name]; exists || IsPipelineTemplateBuiltinVariable(declaration.Name) {
			continue
		}
		if value, ok := EffectiveVariableValue(declaration); ok {
			variables[declaration.Name] = value
		}
	}
	return variables, nil
}

func RepositoryBuiltinVariables(repo model.Repository, triggerRef string) map[string]any {
	return map[string]any{
		"repository_id":   repo.Id,
		"repository_name": repo.Name,
		"repository_code": repo.Code,
		"repository_url":  repo.RepositoryUrl,
		"repository_ref":  triggerRef,
	}
}

func TemplateBuiltinVariables(template model.PipelineTemplate) map[string]any {
	return map[string]any{
		"template_id":      template.Id,
		"template_name":    template.Name,
		"template_version": template.Version,
		"runtime_datetime": time.Now().UTC().Format("20060102-150405"),
	}
}

func EffectiveVariableValue(declaration model.VariableDeclaration) (any, bool) {
	if HasRuntimeValue(declaration.Value) {
		return declaration.Value, true
	}
	if HasRuntimeValue(declaration.Default) {
		return declaration.Default, true
	}
	return nil, false
}

func ValidateRuntimeVariables(variables map[string]any, declarations []model.VariableDeclaration) error {
	missing := make([]string, 0)
	for _, declaration := range declarations {
		if IsPipelineTemplateBuiltinVariable(declaration.Name) {
			continue
		}
		if !HasRuntimeValue(variables[declaration.Name]) {
			missing = append(missing, declaration.Name)
		}
	}
	if len(missing) > 0 {
		return apperror.New(apperror.KindValidation, "Missing variable value: "+strings.Join(missing, ", "))
	}
	return nil
}

func MarshalRuntimeVariableSnapshot(variables map[string]any, declarations []model.VariableDeclaration) (string, error) {
	snapshot := make([]model.VariableDeclaration, 0, len(declarations))
	for _, declaration := range declarations {
		declaration.Value = variables[declaration.Name]
		snapshot = append(snapshot, declaration)
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return "", apperror.Wrap(apperror.KindValidation, "Invalid variables", err)
	}
	return string(data), nil
}

func VariableDeclarations(value string) ([]model.VariableDeclaration, error) {
	if strings.TrimSpace(value) == "" {
		return []model.VariableDeclaration{}, nil
	}
	var declarations []model.VariableDeclaration
	if err := json.Unmarshal([]byte(value), &declarations); err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Invalid variable declarations", err)
	}
	return declarations, nil
}

func VariableDeclarationsFromMaps(values []map[string]any) ([]model.VariableDeclaration, error) {
	data, err := json.Marshal(values)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Invalid variable declarations", err)
	}
	var declarations []model.VariableDeclaration
	if err := json.Unmarshal(data, &declarations); err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Invalid variable declarations", err)
	}
	return declarations, nil
}

func PipelineSnapshotStages(value string) ([]model.StageDefinition, error) {
	if strings.TrimSpace(value) == "" {
		return []model.StageDefinition{}, nil
	}
	var stages []model.StageDefinition
	if err := json.Unmarshal([]byte(value), &stages); err != nil {
		return nil, apperror.New(apperror.KindInternal, "Invalid pipeline snapshot stages")
	}
	return stages, nil
}

func PipelineTemplateVariables(value string) ([]map[string]any, error) {
	if strings.TrimSpace(value) == "" {
		return []map[string]any{}, nil
	}
	var variables []map[string]any
	if err := json.Unmarshal([]byte(value), &variables); err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Invalid pipeline template variables", err)
	}
	return variables, nil
}

func SanitizePipelineTemplateVariables(variables []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(variables))
	for _, variable := range variables {
		name, _ := variable["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" || IsPipelineTemplateBuiltinVariable(name) {
			continue
		}
		source, _ := variable["source"].(string)
		if source == "" {
			source = "template_custom"
		}
		_, hasValue := variable["value"]
		if source != "template_custom" && source != "repository_custom" && (source != "template_stage" || !hasValue || variable["value"] == nil) {
			continue
		}
		copy := map[string]any{}
		maps.Copy(copy, variable)
		copy["name"] = name
		if source == "template_stage" {
			copy["source"] = "template_custom"
		} else {
			copy["source"] = source
		}
		copy["editable"] = true
		if _, exists := copy["secret"]; !exists {
			copy["secret"] = false
		}
		result = append(result, copy)
	}
	return result
}

func ResolveTemplateVariablesFromStageDefinitions(stages []model.StageDefinition, custom []map[string]any) []map[string]any {
	converted := make([]model.PipelineStage, 0, len(stages))
	for _, stage := range stages {
		artifacts, _ := json.Marshal(stage.Artifacts)
		artifactString := string(artifacts)
		converted = append(converted, model.PipelineStage{Name: stage.Name, Script: stage.Script, Artifacts: &artifactString})
	}
	return ResolveTemplateVariables(converted, custom)
}

func ResolveTemplateVariables(stages []model.PipelineStage, custom []map[string]any) []map[string]any {
	extracted := map[string]any{}
	for _, stage := range stages {
		ExtractTemplateVariables(stage.Script, extracted)
		if stage.Artifacts != nil && strings.TrimSpace(*stage.Artifacts) != "" {
			var artifacts []model.ArtifactConfig
			if err := json.Unmarshal([]byte(*stage.Artifacts), &artifacts); err == nil {
				for _, artifact := range artifacts {
					ExtractTemplateVariables(artifact.Reference, extracted)
					ExtractTemplateVariables(artifact.Name, extracted)
					ExtractTemplateVariables(artifact.Command, extracted)
				}
			}
		}
	}
	custom = SanitizePipelineTemplateVariables(custom)
	customByName := make(map[string]map[string]any, len(custom))
	for _, variable := range custom {
		name, _ := variable["name"].(string)
		customByName[name] = variable
	}
	builtinNames := SortedPipelineTemplateBuiltinVariableNames()
	result := []map[string]any{}
	if len(extracted) == 0 {
		for _, name := range builtinNames {
			result = append(result, PipelineTemplateBuiltinVariable(name))
		}
		for _, variable := range custom {
			name, _ := variable["name"].(string)
			if !IsPipelineTemplateBuiltinVariable(name) {
				result = append(result, variable)
			}
		}
		return result
	}
	extractedNames := make([]string, 0, len(extracted))
	for name := range extracted {
		extractedNames = append(extractedNames, name)
	}
	sort.Strings(extractedNames)
	for _, name := range extractedNames {
		if IsPipelineTemplateBuiltinVariable(name) {
			result = append(result, PipelineTemplateBuiltinVariable(name))
			continue
		}
		if existing, ok := customByName[name]; ok {
			if _, exists := existing["default"]; !exists && extracted[name] != nil {
				existing["default"] = extracted[name]
			}
			result = append(result, existing)
			continue
		}
		result = append(result, map[string]any{"name": name, "description": "", "default": extracted[name], "value": nil, "secret": false, "source": "template_stage", "editable": true})
	}
	return result
}

func ExtractTemplateVariables(text string, found map[string]any) {
	for _, match := range templateVariablePattern.FindAllStringSubmatch(text, -1) {
		name := match[1]
		defaultValue := any(nil)
		if len(match) > 2 && match[2] != "" {
			defaultValue = match[2]
		}
		if current, exists := found[name]; !exists || current == nil && defaultValue != nil {
			found[name] = defaultValue
		}
	}
}

func HasRuntimeValue(value any) bool {
	if value == nil {
		return false
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) != ""
	}
	return true
}

func IsPipelineTemplateBuiltinVariable(name string) bool {
	_, ok := PipelineTemplateBuiltinVariableSpecs()[name]
	return ok
}

func SortedPipelineTemplateBuiltinVariableNames() []string {
	names := make([]string, 0, len(PipelineTemplateBuiltinVariableSpecs()))
	for name := range PipelineTemplateBuiltinVariableSpecs() {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func PipelineTemplateBuiltinVariable(name string) map[string]any {
	spec := PipelineTemplateBuiltinVariableSpecs()[name]
	return map[string]any{"name": name, "description": spec, "default": nil, "value": nil, "secret": false, "source": "template", "editable": false}
}

func PipelineTemplateBuiltinVariableSpecs() map[string]string {
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
