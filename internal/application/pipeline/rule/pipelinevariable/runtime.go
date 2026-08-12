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

// RuntimeVariableDeclarations builds the declarations used by a run. Snapshot
// declarations are historical data; callers must provide the current
// repository, pipeline and stage definitions.
func RuntimeVariableDeclarations(repo model.Repository, pipeline model.Pipeline, stages []model.StageDefinition) ([]model.VariableDeclaration, error) {
	repositoryVariables, err := repositoryVariableDeclarations(repo.VariableOverrides)
	if err != nil {
		return nil, err
	}
	pipelineVariables, err := PipelineVariables(pipeline.VariableDeclarations)
	if err != nil {
		return nil, err
	}
	stageVariables := extractStageVariableDeclarations(stages)

	byName := make(map[string]model.VariableDeclaration)
	// Repository custom values have the highest persisted-config priority.
	for _, declaration := range repositoryVariables {
		byName[declaration.Name] = declaration
	}
	for _, raw := range SanitizePipelineVariables(pipelineVariables) {
		declaration, err := variableDeclarationFromMap(raw, "pipeline")
		if err != nil {
			return nil, err
		}
		if current, exists := byName[declaration.Name]; exists {
			mergeFallback(&current, declaration)
			byName[declaration.Name] = current
			continue
		}
		byName[declaration.Name] = declaration
	}
	for _, declaration := range stageVariables {
		if current, exists := byName[declaration.Name]; exists {
			mergeFallback(&current, declaration)
			byName[declaration.Name] = current
			continue
		}
		byName[declaration.Name] = declaration
	}

	result := make([]model.VariableDeclaration, 0, len(byName)+len(PipelineBuiltinVariableSpecs()))
	for _, name := range sortedDeclarationNames(byName) {
		result = append(result, byName[name])
	}
	result = append(result,
		model.VariableDeclaration{Name: "repository_ref", Description: PipelineBuiltinVariableSpecs()["repository_ref"], Default: repo.DefaultBranch, Source: "runtime", Editable: true},
	)
	for _, name := range SortedPipelineBuiltinVariableNames() {
		if name == "repository_ref" {
			continue
		}
		result = append(result, model.VariableDeclaration{Name: name, Description: PipelineBuiltinVariableSpecs()[name], Source: "system", Editable: false})
	}
	return result, nil
}

// ResolveRuntimeVariables applies the complete form -> repository -> pipeline
// -> stage-default chain and injects the immutable system context.
func ResolveRuntimeVariables(repo model.Repository, pipeline model.Pipeline, stages []model.StageDefinition, form map[string]string, requireCompleteForm bool) ([]model.VariableDeclaration, map[string]any, error) {
	declarations, err := RuntimeVariableDeclarations(repo, pipeline, stages)
	if err != nil {
		return nil, nil, err
	}
	declarationByName := make(map[string]model.VariableDeclaration, len(declarations))
	for _, declaration := range declarations {
		declarationByName[declaration.Name] = declaration
	}
	for name := range form {
		if _, ok := declarationByName[name]; !ok {
			return nil, nil, apperror.New(apperror.KindValidation, "Unknown variable: "+name)
		}
		if declarationByName[name].Source == "system" {
			return nil, nil, apperror.New(apperror.KindValidation, "System variable cannot be overridden: "+name)
		}
	}
	if requireCompleteForm {
		for _, declaration := range declarations {
			if declaration.Source == "system" {
				continue
			}
			if value, ok := form[declaration.Name]; ok && HasRuntimeValue(value) {
				continue
			}
			if declaration.Name == "repository_ref" {
				return nil, nil, apperror.New(apperror.KindValidation, "repository_ref is required")
			}
			return nil, nil, apperror.New(apperror.KindValidation, "Missing variable value: "+declaration.Name)
		}
	}

	values := make(map[string]any, len(declarations))
	for _, declaration := range declarations {
		switch declaration.Source {
		case "system":
			values[declaration.Name] = systemVariableValue(repo, declaration.Name)
		default:
			if value, ok := form[declaration.Name]; ok && HasRuntimeValue(value) {
				values[declaration.Name] = value
			} else if value, ok := EffectiveVariableValue(declaration); ok {
				values[declaration.Name] = value
			}
		}
	}
	for _, declaration := range declarations {
		if requireCompleteForm && declaration.Source != "system" && !HasRuntimeValue(values[declaration.Name]) {
			return nil, nil, apperror.New(apperror.KindValidation, "Missing variable value: "+declaration.Name)
		}
	}
	return declarations, values, nil
}

func ResolveRuntimeVariablesFromPipelineStages(repo model.Repository, pipeline model.Pipeline, stages []model.PipelineStage, form map[string]string, requireCompleteForm bool) ([]model.VariableDeclaration, map[string]any, error) {
	definitions := make([]model.StageDefinition, 0, len(stages))
	for _, stage := range stages {
		definition := model.StageDefinition{Id: stage.Id, Name: stage.Name, Image: stage.Image, Script: stage.Script, Description: stage.Description}
		if stage.Artifacts != nil && strings.TrimSpace(*stage.Artifacts) != "" {
			if err := json.Unmarshal([]byte(*stage.Artifacts), &definition.Artifacts); err != nil {
				return nil, nil, apperror.Wrap(apperror.KindInternal, "Invalid pipeline stage artifacts", err)
			}
		}
		if stage.DependsOn != nil && strings.TrimSpace(*stage.DependsOn) != "" {
			if err := json.Unmarshal([]byte(*stage.DependsOn), &definition.DependsOn); err != nil {
				return nil, nil, apperror.Wrap(apperror.KindInternal, "Invalid pipeline stage dependencies", err)
			}
		}
		if stage.SortOrder != nil {
			definition.SortOrder = *stage.SortOrder
		}
		definitions = append(definitions, definition)
	}
	return ResolveRuntimeVariables(repo, pipeline, definitions, form, requireCompleteForm)
}

func UnmarshalRuntimeVariableSnapshot(value string) ([]model.VariableDeclaration, map[string]any, error) {
	declarations, err := VariableDeclarations(value)
	if err != nil {
		return nil, nil, err
	}
	values := make(map[string]any, len(declarations))
	for _, declaration := range declarations {
		if declaration.Name == "" || declaration.Source == "" {
			return nil, nil, apperror.New(apperror.KindValidation, "Invalid pipeline run variable declaration")
		}
		if _, exists := values[declaration.Name]; exists {
			return nil, nil, apperror.New(apperror.KindValidation, "Duplicate pipeline run variable: "+declaration.Name)
		}
		if !HasRuntimeValue(declaration.Value) {
			return nil, nil, apperror.New(apperror.KindValidation, "Missing pipeline run variable: "+declaration.Name)
		}
		values[declaration.Name] = declaration.Value
	}
	return declarations, values, nil
}

func repositoryVariableDeclarations(value string) ([]model.VariableDeclaration, error) {
	variables, err := PipelineVariables(value)
	if err != nil {
		return nil, err
	}
	result := make([]model.VariableDeclaration, 0, len(variables))
	for _, raw := range variables {
		name, _ := raw["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" || IsPipelineBuiltinVariable(name) {
			continue
		}
		declaration, err := variableDeclarationFromMap(raw, "repository")
		if err != nil {
			return nil, err
		}
		result = append(result, declaration)
	}
	return result, nil
}

func extractStageVariableDeclarations(stages []model.StageDefinition) []model.VariableDeclaration {
	found := map[string]any{}
	for _, stage := range stages {
		extractPipelineVariables(stage.Script, found)
		for _, artifact := range stage.Artifacts {
			extractPipelineVariables(artifact.Reference, found)
			extractPipelineVariables(artifact.Name, found)
			extractPipelineVariables(artifact.Command, found)
		}
	}
	names := make([]string, 0, len(found))
	for name := range found {
		if !IsPipelineBuiltinVariable(name) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	result := make([]model.VariableDeclaration, 0, len(names))
	for _, name := range names {
		result = append(result, model.VariableDeclaration{Name: name, Default: found[name], Source: "pipeline_stage", Editable: true})
	}
	return result
}

func variableDeclarationFromMap(raw map[string]any, source string) (model.VariableDeclaration, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return model.VariableDeclaration{}, apperror.Wrap(apperror.KindInternal, "Invalid variable declaration", err)
	}
	var declaration model.VariableDeclaration
	if err := json.Unmarshal(data, &declaration); err != nil {
		return model.VariableDeclaration{}, apperror.Wrap(apperror.KindInternal, "Invalid variable declaration", err)
	}
	declaration.Name = strings.TrimSpace(declaration.Name)
	declaration.Source = source
	declaration.Editable = true
	return declaration, nil
}

func sortedDeclarationNames(values map[string]model.VariableDeclaration) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func mergeFallback(high *model.VariableDeclaration, low model.VariableDeclaration) {
	if HasRuntimeValue(high.Value) || HasRuntimeValue(high.Default) {
		return
	}
	if HasRuntimeValue(low.Value) {
		high.Value = low.Value
		return
	}
	if HasRuntimeValue(low.Default) {
		high.Default = low.Default
	}
}

func systemVariableValue(repo model.Repository, name string) any {
	switch name {
	case "repository_code":
		return repo.Code
	case "repository_url":
		return repo.RepositoryUrl
	case "runtime_datetime":
		return time.Now().UTC().Format("20060102-150405")
	default:
		return nil
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
	result := make([]map[string]any, 0, len(extracted)+len(custom)+len(PipelineBuiltinVariableSpecs()))
	for _, name := range SortedPipelineBuiltinVariableNames() {
		result = append(result, PipelineBuiltinVariable(name))
	}
	names := make([]string, 0, len(extracted))
	for name := range extracted {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if IsPipelineBuiltinVariable(name) {
			continue
		}
		if existing, ok := customByName[name]; ok {
			copy := map[string]any{}
			maps.Copy(copy, existing)
			if _, exists := existing["default"]; !exists && extracted[name] != nil {
				copy["default"] = extracted[name]
			}
			result = append(result, copy)
			continue
		}
		result = append(result, map[string]any{"name": name, "description": "", "default": extracted[name], "value": nil, "secret": false, "source": "pipeline_stage", "editable": true})
	}
	for _, variable := range custom {
		name, _ := variable["name"].(string)
		if _, exists := extracted[name]; !exists {
			result = append(result, variable)
		}
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
		"repository_code": "运行时注入: 当前仓库编码", "repository_url": "运行时注入: 当前仓库地址",
		"repository_ref":   "运行时注入: 当前分支",
		"runtime_datetime": "运行时注入: 流水线启动时间 (UTC, 格式 YYYYmmdd-HHmmss)",
	}
}
