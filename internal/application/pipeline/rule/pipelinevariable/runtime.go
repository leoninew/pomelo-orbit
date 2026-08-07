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

var pipelineVariablePattern = regexp.MustCompile(`\{\{\s*([A-Za-z][A-Za-z0-9_]*)(?:\s*\|\s*default\s*:\s*['"]([^'"]*)['"])?\s*\}\}`)

func CompleteSnapshotVariableDeclarations(snapshot model.PipelineSnapshot, pipeline model.Pipeline) ([]model.VariableDeclaration, error) {
	variables, err := VariableDeclarations(snapshot.VariablesSnapshot)
	if err != nil || len(variables) > 0 {
		return variables, err
	}
	var stages []model.StageDefinition
	if err := json.Unmarshal([]byte(snapshot.StagesSnapshot), &stages); err != nil {
		return nil, apperror.New(apperror.KindInternal, "Invalid pipeline snapshot stages")
	}
	custom, err := PipelineVariables(pipeline.VariableDeclarations)
	if err != nil {
		return nil, err
	}
	return VariableDeclarationsFromMaps(ResolvePipelineVariablesFromStageDefinitions(stages, custom))
}

func BuildRuntimeVariables(repo model.Repository, pipeline model.Pipeline, triggerRef string, runtimeOverrides map[string]string, declarations []model.VariableDeclaration) (map[string]any, error) {
	variables := map[string]any{}
	maps.Copy(variables, RepositoryBuiltinVariables(repo, triggerRef))
	maps.Copy(variables, PipelineBuiltinVariables(pipeline))
	for name, value := range runtimeOverrides {
		name = strings.TrimSpace(name)
		if name != "" && !IsPipelineBuiltinVariable(name) {
			variables[name] = value
		}
	}
	for _, source := range []string{repo.VariableOverrides, pipeline.VariableDeclarations} {
		configured, err := VariableDeclarations(source)
		if err != nil {
			return nil, err
		}
		for _, declaration := range configured {
			if _, exists := variables[declaration.Name]; !exists && !IsPipelineBuiltinVariable(declaration.Name) {
				if value, ok := EffectiveVariableValue(declaration); ok {
					variables[declaration.Name] = value
				}
			}
		}
	}
	for _, declaration := range declarations {
		if _, exists := variables[declaration.Name]; !exists && !IsPipelineBuiltinVariable(declaration.Name) {
			if value, ok := EffectiveVariableValue(declaration); ok {
				variables[declaration.Name] = value
			}
		}
	}
	return variables, nil
}

func RepositoryBuiltinVariables(repo model.Repository, triggerRef string) map[string]any {
	return map[string]any{
		"repository_id": repo.Id, "repository_name": repo.Name, "repository_code": repo.Code,
		"repository_url": repo.RepositoryUrl, "repository_ref": triggerRef,
	}
}

func PipelineBuiltinVariables(pipeline model.Pipeline) map[string]any {
	return map[string]any{
		"pipeline_id": pipeline.Id, "pipeline_name": pipeline.Name, "pipeline_version": pipeline.Version,
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
		if !IsPipelineBuiltinVariable(declaration.Name) && !HasRuntimeValue(variables[declaration.Name]) {
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

// ResolvePipelineVariableDeclarations combines pipeline custom declarations
// with variables extracted from the pipeline stages.
func ResolvePipelineVariableDeclarations(stages []model.PipelineStage, pipelineVariables string) ([]model.VariableDeclaration, error) {
	custom, err := PipelineVariables(pipelineVariables)
	if err != nil {
		return nil, err
	}
	return VariableDeclarationsFromMaps(ResolvePipelineVariables(stages, custom))
}

func PipelineVariables(value string) ([]map[string]any, error) {
	if strings.TrimSpace(value) == "" {
		return []map[string]any{}, nil
	}
	var variables []map[string]any
	if err := json.Unmarshal([]byte(value), &variables); err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Invalid pipeline variables", err)
	}
	return variables, nil
}

func SanitizePipelineVariables(variables []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(variables))
	for _, variable := range variables {
		name, _ := variable["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" || IsPipelineBuiltinVariable(name) {
			continue
		}
		source, _ := variable["source"].(string)
		if source == "" {
			source = "pipeline_custom"
		}
		if source != "pipeline_custom" && source != "pipeline_stage" {
			continue
		}
		copy := map[string]any{}
		maps.Copy(copy, variable)
		copy["name"], copy["source"], copy["editable"] = name, "pipeline_custom", true
		if _, exists := copy["secret"]; !exists {
			copy["secret"] = false
		}
		result = append(result, copy)
	}
	return result
}

func ResolvePipelineVariablesFromStageDefinitions(stages []model.StageDefinition, custom []map[string]any) []map[string]any {
	converted := make([]model.PipelineStage, 0, len(stages))
	for _, stage := range stages {
		artifacts, _ := json.Marshal(stage.Artifacts)
		artifactData := string(artifacts)
		converted = append(converted, model.PipelineStage{Name: stage.Name, Script: stage.Script, Artifacts: &artifactData})
	}
	return ResolvePipelineVariables(converted, custom)
}

func ResolvePipelineVariables(stages []model.PipelineStage, custom []map[string]any) []map[string]any {
	extracted := map[string]any{}
	for _, stage := range stages {
		extractPipelineVariables(stage.Script, extracted)
		if stage.Artifacts == nil || strings.TrimSpace(*stage.Artifacts) == "" {
			continue
		}
		var artifacts []model.ArtifactConfig
		if json.Unmarshal([]byte(*stage.Artifacts), &artifacts) != nil {
			continue
		}
		for _, artifact := range artifacts {
			extractPipelineVariables(artifact.Reference, extracted)
			extractPipelineVariables(artifact.Name, extracted)
			extractPipelineVariables(artifact.Command, extracted)
		}
	}
	custom = SanitizePipelineVariables(custom)
	customByName := map[string]map[string]any{}
	for _, variable := range custom {
		name, _ := variable["name"].(string)
		customByName[name] = variable
	}
	result := make([]map[string]any, 0, len(extracted)+len(custom))
	if len(extracted) == 0 {
		for _, name := range SortedPipelineBuiltinVariableNames() {
			result = append(result, PipelineBuiltinVariable(name))
		}
		return append(result, custom...)
	}
	names := make([]string, 0, len(extracted))
	for name := range extracted {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if IsPipelineBuiltinVariable(name) {
			result = append(result, PipelineBuiltinVariable(name))
			continue
		}
		if existing, ok := customByName[name]; ok {
			if _, exists := existing["default"]; !exists && extracted[name] != nil {
				existing["default"] = extracted[name]
			}
			result = append(result, existing)
			continue
		}
		result = append(result, map[string]any{"name": name, "description": "", "default": extracted[name], "value": nil, "secret": false, "source": "pipeline_stage", "editable": true})
	}
	return result
}

func extractPipelineVariables(text string, found map[string]any) {
	for _, match := range pipelineVariablePattern.FindAllStringSubmatch(text, -1) {
		value := any(nil)
		if len(match) > 2 && match[2] != "" {
			value = match[2]
		}
		if current, exists := found[match[1]]; !exists || current == nil && value != nil {
			found[match[1]] = value
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

func IsPipelineBuiltinVariable(name string) bool {
	_, ok := PipelineBuiltinVariableSpecs()[name]
	return ok
}

func SortedPipelineBuiltinVariableNames() []string {
	names := make([]string, 0, len(PipelineBuiltinVariableSpecs()))
	for name := range PipelineBuiltinVariableSpecs() {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func PipelineBuiltinVariable(name string) map[string]any {
	return map[string]any{"name": name, "description": PipelineBuiltinVariableSpecs()[name], "default": nil, "value": nil, "secret": false, "source": "pipeline", "editable": false}
}

func PipelineBuiltinVariableSpecs() map[string]string {
	return map[string]string{
		"repository_id": "运行时注入: 当前仓库 ID", "repository_name": "运行时注入: 当前仓库名称",
		"repository_code": "运行时注入: 当前仓库编码", "repository_url": "运行时注入: 当前仓库地址",
		"repository_ref": "运行时注入: 当前分支", "pipeline_id": "运行时注入: 当前流水线 ID",
		"pipeline_name": "运行时注入: 当前流水线名称", "pipeline_version": "运行时注入: 当前流水线版本",
		"runtime_datetime": "运行时注入: 流水线启动时间 (UTC, 格式 YYYYmmdd-HHmmss)",
	}
}
