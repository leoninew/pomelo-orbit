package pipelinevariable

import (
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"sort"
	"strings"
	"time"

	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	templatex "github.com/leoninew/pomelo-orbit/internal/common/template"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

var pipelineVariableNamePattern = regexp.MustCompile("^[A-Za-z][A-Za-z0-9_]*$")

type RuntimeVariableOverrides struct {
	Global map[string]string
	Stage  map[string]map[string]string
}

type RuntimeVariables struct {
	Global map[string]any
	Stage  map[string]map[string]any
}

func (r RuntimeVariables) ValuesForStage(stage model.StageDefinition) map[string]any {
	values := make(map[string]any, len(r.Global)+len(r.Stage[stageScopeID(stage)]))
	maps.Copy(values, r.Global)
	maps.Copy(values, r.Stage[stageScopeID(stage)])
	return values
}

func stageScopeID(stage model.StageDefinition) string {
	if strings.TrimSpace(stage.Id) != "" {
		return stage.Id
	}
	return stage.Name
}

func declarationScopeKey(name, stageID string) string {
	return name + "\x00" + stageID
}

func declarationScopeID(declaration model.VariableDeclaration) string {
	return declaration.StageId
}

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
	pipelineVariables, err = NormalizePipelineVariables(pipelineVariables)
	if err != nil {
		return nil, err
	}
	if err := ValidatePipelineVariableScopes(pipelineVariables, stages); err != nil {
		return nil, err
	}
	stageVariables, err := extractStageVariableDeclarations(stages)
	if err != nil {
		return nil, err
	}

	repositoryByName := make(map[string]model.VariableDeclaration, len(repositoryVariables))
	for _, declaration := range repositoryVariables {
		repositoryByName[declaration.Name] = declaration
	}
	pipelineGlobals := make(map[string]model.VariableDeclaration)
	pipelineStages := make(map[string]model.VariableDeclaration)
	for _, raw := range pipelineVariables {
		declaration, err := variableDeclarationFromMap(raw, "pipeline")
		if err != nil {
			return nil, err
		}
		if declaration.StageId == "" {
			pipelineGlobals[declaration.Name] = declaration
		} else {
			pipelineStages[declarationScopeKey(declaration.Name, declaration.StageId)] = declaration
		}
	}

	result := make([]model.VariableDeclaration, 0, len(repositoryVariables)+len(pipelineVariables)+len(stageVariables)+len(PipelineBuiltinVariableSpecs()))
	globalNames := make(map[string]struct{}, len(repositoryByName)+len(pipelineGlobals))
	for name := range repositoryByName {
		globalNames[name] = struct{}{}
	}
	for name := range pipelineGlobals {
		globalNames[name] = struct{}{}
	}
	for _, name := range sortedStringSet(globalNames) {
		if declaration, ok := repositoryByName[name]; ok {
			result = append(result, declaration)
			continue
		}
		result = append(result, pipelineGlobals[name])
	}
	for _, declaration := range stageVariables {
		if configured, ok := pipelineStages[declarationScopeKey(declaration.Name, declaration.StageId)]; ok {
			configured.StageId, configured.StageName = declaration.StageId, declaration.StageName
			configured.StageDefaults = declaration.StageDefaults
			result = append(result, configured)
			continue
		}
		result = append(result, declaration)
	}
	unmatchedPipelineStages := make([]model.VariableDeclaration, 0)
	for _, declaration := range pipelineStages {
		if !containsStageVariable(stageVariables, declaration.Name, declaration.StageId) {
			unmatchedPipelineStages = append(unmatchedPipelineStages, declaration)
		}
	}
	sort.Slice(unmatchedPipelineStages, func(i, j int) bool {
		if unmatchedPipelineStages[i].Name != unmatchedPipelineStages[j].Name {
			return unmatchedPipelineStages[i].Name < unmatchedPipelineStages[j].Name
		}
		return unmatchedPipelineStages[i].StageId < unmatchedPipelineStages[j].StageId
	})
	result = append(result, unmatchedPipelineStages...)
	result = append(result, model.VariableDeclaration{Name: "repository_ref", Description: PipelineBuiltinVariableSpecs()["repository_ref"], Default: repo.DefaultBranch, Source: "runtime", Editable: false})
	for _, name := range SortedPipelineBuiltinVariableNames() {
		if name == "repository_ref" {
			continue
		}
		result = append(result, model.VariableDeclaration{Name: name, Description: PipelineBuiltinVariableSpecs()[name], Source: "system", Editable: false})
	}
	return result, nil
}

func containsStageVariable(declarations []model.VariableDeclaration, name, stageID string) bool {
	for _, declaration := range declarations {
		if declaration.Name == name && declaration.StageId == stageID {
			return true
		}
	}
	return false
}

// ResolveRuntimeVariables applies internal retry overrides -> repository ->
// pipeline Stage -> pipeline global -> the current Liquid default.
func ResolveRuntimeVariables(repo model.Repository, pipeline model.Pipeline, stages []model.StageDefinition, overrides RuntimeVariableOverrides) ([]model.VariableDeclaration, RuntimeVariables, error) {
	declarations, err := RuntimeVariableDeclarations(repo, pipeline, stages)
	if err != nil {
		return nil, RuntimeVariables{}, err
	}
	knownNames := map[string]struct{}{}
	knownStages := map[string]struct{}{}
	for _, declaration := range declarations {
		knownNames[declaration.Name] = struct{}{}
		if declaration.StageId != "" {
			knownStages[declaration.StageId] = struct{}{}
		}
	}
	for name := range overrides.Global {
		if _, ok := knownNames[name]; !ok {
			return nil, RuntimeVariables{}, apperror.New(apperror.KindValidation, "Unknown variable: "+name)
		}
		if name != "repository_ref" && IsPipelineBuiltinVariable(name) {
			return nil, RuntimeVariables{}, apperror.New(apperror.KindValidation, "System variable cannot be overridden: "+name)
		}
	}
	for stageID, values := range overrides.Stage {
		if _, ok := knownStages[stageID]; !ok {
			return nil, RuntimeVariables{}, apperror.New(apperror.KindValidation, "Unknown pipeline stage: "+stageID)
		}
		for name := range values {
			if _, ok := knownNames[name]; !ok {
				return nil, RuntimeVariables{}, apperror.New(apperror.KindValidation, "Unknown variable: "+name)
			}
			if IsPipelineBuiltinVariable(name) {
				return nil, RuntimeVariables{}, apperror.New(apperror.KindValidation, "System variable cannot be overridden: "+name)
			}
		}
	}

	repositoryVariables, err := repositoryVariableDeclarations(repo.VariableOverrides)
	if err != nil {
		return nil, RuntimeVariables{}, err
	}
	pipelineVariables, err := PipelineVariables(pipeline.VariableDeclarations)
	if err != nil {
		return nil, RuntimeVariables{}, err
	}
	pipelineVariables, err = NormalizePipelineVariables(pipelineVariables)
	if err != nil {
		return nil, RuntimeVariables{}, err
	}
	if err := ValidatePipelineVariableScopes(pipelineVariables, stages); err != nil {
		return nil, RuntimeVariables{}, err
	}
	repositoryByName := make(map[string]model.VariableDeclaration, len(repositoryVariables))
	for _, declaration := range repositoryVariables {
		repositoryByName[declaration.Name] = declaration
	}
	pipelineGlobalByName := make(map[string]model.VariableDeclaration)
	pipelineStageByKey := make(map[string]model.VariableDeclaration)
	for _, raw := range pipelineVariables {
		declaration, err := variableDeclarationFromMap(raw, "pipeline")
		if err != nil {
			return nil, RuntimeVariables{}, err
		}
		if declaration.StageId == "" {
			pipelineGlobalByName[declaration.Name] = declaration
		} else {
			pipelineStageByKey[declarationScopeKey(declaration.Name, declaration.StageId)] = declaration
		}
	}

	runtime := RuntimeVariables{Global: make(map[string]any), Stage: make(map[string]map[string]any)}
	for name, declaration := range repositoryByName {
		if value, ok := EffectiveVariableValue(declaration); ok {
			runtime.Global[name] = value
		}
	}
	for name, declaration := range pipelineGlobalByName {
		if _, exists := runtime.Global[name]; exists {
			continue
		}
		if value, ok := EffectiveVariableValue(declaration); ok {
			runtime.Global[name] = value
		}
	}
	for name, value := range overrides.Global {
		if HasRuntimeValue(value) {
			runtime.Global[name] = value
		}
	}
	runtime.Global["repository_code"] = systemVariableValue(repo, "repository_code")
	runtime.Global["repository_url"] = systemVariableValue(repo, "repository_url")
	runtime.Global["runtime_datetime"] = systemVariableValue(repo, "runtime_datetime")
	runtime.Global["repository_ref"] = repo.DefaultBranch
	if value, ok := overrides.Global["repository_ref"]; ok && HasRuntimeValue(value) {
		runtime.Global["repository_ref"] = value
	}

	for _, stage := range stages {
		stageID := stageScopeID(stage)
		stageValues := make(map[string]any)
		for name, value := range runtime.Global {
			if name == "repository_code" || name == "repository_url" || name == "runtime_datetime" || name == "repository_ref" {
				stageValues[name] = value
			}
		}
		for name := range knownNames {
			if value, ok := overrides.Stage[stageID][name]; ok && HasRuntimeValue(value) {
				stageValues[name] = value
				continue
			}
			if value, ok := overrides.Global[name]; ok && HasRuntimeValue(value) {
				stageValues[name] = value
				continue
			}
			if repository, ok := repositoryByName[name]; ok {
				if value, ok := EffectiveVariableValue(repository); ok {
					stageValues[name] = value
					continue
				}
			}
			if configured, ok := pipelineStageByKey[declarationScopeKey(name, stageID)]; ok {
				if value, ok := EffectiveVariableValue(configured); ok {
					stageValues[name] = value
					continue
				}
			}
			if _, exists := stageValues[name]; !exists {
				if value, ok := runtime.Global[name]; ok {
					stageValues[name] = value
				}
			}
		}
		runtime.Stage[stageID] = stageValues
	}
	if err := validateStageTemplates(stages, runtime); err != nil {
		return nil, RuntimeVariables{}, err
	}
	for _, declaration := range declarations {
		if declaration.Source == "system" || declaration.StageId != "" {
			continue
		}
		if declaration.Name == "repository_ref" {
			continue
		}
		if !HasRuntimeValue(runtime.Global[declaration.Name]) {
			return nil, RuntimeVariables{}, apperror.New(apperror.KindValidation, "Missing variable value: "+declaration.Name)
		}
	}
	return declarations, runtime, nil
}

func ResolveRuntimeVariablesFromPipelineStages(repo model.Repository, pipeline model.Pipeline, stages []model.PipelineStage, overrides RuntimeVariableOverrides) ([]model.VariableDeclaration, RuntimeVariables, error) {
	definitions := make([]model.StageDefinition, 0, len(stages))
	for _, stage := range stages {
		definition := model.StageDefinition{Id: stage.Id, Name: stage.Name, Image: stage.Image, Script: stage.Script, Description: stage.Description}
		if stage.Artifacts != nil && strings.TrimSpace(*stage.Artifacts) != "" {
			if err := json.Unmarshal([]byte(*stage.Artifacts), &definition.Artifacts); err != nil {
				return nil, RuntimeVariables{}, apperror.Wrap(apperror.KindInternal, "Invalid pipeline stage artifacts", err)
			}
		}
		if stage.DependsOn != nil && strings.TrimSpace(*stage.DependsOn) != "" {
			if err := json.Unmarshal([]byte(*stage.DependsOn), &definition.DependsOn); err != nil {
				return nil, RuntimeVariables{}, apperror.Wrap(apperror.KindInternal, "Invalid pipeline stage dependencies", err)
			}
		}
		if stage.SortOrder != nil {
			definition.SortOrder = *stage.SortOrder
		}
		definitions = append(definitions, definition)
	}
	return ResolveRuntimeVariables(repo, pipeline, definitions, overrides)
}

func UnmarshalRuntimeVariableSnapshot(value string) ([]model.VariableDeclaration, RuntimeVariables, error) {
	declarations, err := VariableDeclarations(value)
	if err != nil {
		return nil, RuntimeVariables{}, err
	}
	runtime := RuntimeVariables{Global: make(map[string]any), Stage: make(map[string]map[string]any)}
	seen := make(map[string]struct{}, len(declarations))
	for _, declaration := range declarations {
		if declaration.Name == "" || declaration.Source == "" {
			return nil, RuntimeVariables{}, apperror.New(apperror.KindValidation, "Invalid pipeline run variable declaration")
		}
		key := declarationScopeKey(declaration.Name, declarationScopeID(declaration))
		if _, exists := seen[key]; exists {
			return nil, RuntimeVariables{}, apperror.New(apperror.KindValidation, "Duplicate pipeline run variable: "+key)
		}
		seen[key] = struct{}{}
		if declaration.StageId != "" {
			if HasRuntimeValue(declaration.Value) {
				if runtime.Stage[declaration.StageId] == nil {
					runtime.Stage[declaration.StageId] = make(map[string]any)
				}
				runtime.Stage[declaration.StageId][declaration.Name] = declaration.Value
			}
			if !HasRuntimeValue(declaration.Value) && len(declaration.StageDefaults) == 0 {
				return nil, RuntimeVariables{}, apperror.New(apperror.KindValidation, "Missing pipeline run variable: "+declaration.Name+" in stage "+declaration.StageName)
			}
			continue
		}
		if !HasRuntimeValue(declaration.Value) && declaration.Source != "system" && declaration.Name != "repository_ref" {
			return nil, RuntimeVariables{}, apperror.New(apperror.KindValidation, "Missing pipeline run variable: "+declaration.Name)
		}
		if HasRuntimeValue(declaration.Value) {
			runtime.Global[declaration.Name] = declaration.Value
		}
	}
	return declarations, runtime, nil
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

func extractStageVariableDeclarations(stages []model.StageDefinition) ([]model.VariableDeclaration, error) {
	found := map[string]*model.VariableDeclaration{}
	for _, stage := range stages {
		if err := collectStageVariableReferences(stage, stage.Script, "script", found); err != nil {
			return nil, err
		}
		for index, artifact := range stage.Artifacts {
			if err := collectStageVariableReferences(stage, artifact.Reference, fmt.Sprintf("artifacts[%d].reference", index), found); err != nil {
				return nil, err
			}
			if err := collectStageVariableReferences(stage, artifact.Name, fmt.Sprintf("artifacts[%d].name", index), found); err != nil {
				return nil, err
			}
			if err := collectStageVariableReferences(stage, artifact.Command, fmt.Sprintf("artifacts[%d].command", index), found); err != nil {
				return nil, err
			}
		}
	}
	result := make([]model.VariableDeclaration, 0, len(found))
	for _, declaration := range found {
		result = append(result, *declaration)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].StageId < result[j].StageId
	})
	return result, nil
}

func collectStageVariableReferences(stage model.StageDefinition, text, field string, found map[string]*model.VariableDeclaration) error {
	references, err := templatex.ExtractPipelineVariableReferences(text)
	if err != nil {
		return apperror.New(apperror.KindValidation, fmt.Sprintf("Stage %s %s: %v", stage.Name, field, err))
	}
	stageID := stageScopeID(stage)
	for _, reference := range references {
		if IsPipelineBuiltinVariable(reference.Name) {
			continue
		}
		key := declarationScopeKey(reference.Name, stageID)
		declaration, exists := found[key]
		if !exists {
			declaration = &model.VariableDeclaration{Name: reference.Name, Source: "pipeline_stage", Editable: true, StageId: stageID, StageName: stage.Name}
			found[key] = declaration
		}
		if reference.HasDefault {
			declaration.StageDefaults = append(declaration.StageDefaults, model.StageVariableDefault{StageId: stageID, StageName: stage.Name, Default: reference.Default})
		}
	}
	return nil
}

func validateStageTemplates(stages []model.StageDefinition, runtime RuntimeVariables) error {
	for _, stage := range stages {
		values := runtime.ValuesForStage(stage)
		if err := validateStageTemplateField(stage, "script", stage.Script, values); err != nil {
			return err
		}
		for index, artifact := range stage.Artifacts {
			for _, field := range []struct{ name, text string }{
				{name: fmt.Sprintf("artifacts[%d].reference", index), text: artifact.Reference},
				{name: fmt.Sprintf("artifacts[%d].name", index), text: artifact.Name},
				{name: fmt.Sprintf("artifacts[%d].command", index), text: artifact.Command},
			} {
				if err := validateStageTemplateField(stage, field.name, field.text, values); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateStageTemplateField(stage model.StageDefinition, field, text string, values map[string]any) error {
	if _, err := templatex.ExtractPipelineVariableReferences(text); err != nil {
		return apperror.New(apperror.KindValidation, fmt.Sprintf("Stage %s %s: %v", stage.Name, field, err))
	}
	if _, err := templatex.Render(text, values); err != nil {
		return apperror.New(apperror.KindValidation, fmt.Sprintf("Stage %s %s: %v", stage.Name, field, err))
	}
	return nil
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
	declaration.StageId = strings.TrimSpace(declaration.StageId)
	declaration.StageName = strings.TrimSpace(declaration.StageName)
	declaration.Source = source
	declaration.Editable = true
	declaration.StageDefaults = nil
	return declaration, nil
}

func sortedStringSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
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

func MarshalRuntimeVariableSnapshot(runtime RuntimeVariables, declarations []model.VariableDeclaration) (string, error) {
	snapshot := make([]model.VariableDeclaration, 0, len(declarations))
	for _, declaration := range declarations {
		declaration.Value = nil
		if declaration.StageId != "" {
			declaration.Value = runtime.Stage[declaration.StageId][declaration.Name]
		} else {
			declaration.Value = runtime.Global[declaration.Name]
		}
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

// ResolvePipelineVariableDeclarations returns the variables displayed by an
// application pipeline: its managed declarations plus read-only built-ins.
func ResolvePipelineVariableDeclarations(stages []model.PipelineStage, pipelineVariables string) ([]model.VariableDeclaration, error) {
	custom, err := PipelineVariables(pipelineVariables)
	if err != nil {
		return nil, err
	}
	variables, err := ResolvePipelineVariables(stages, custom)
	if err != nil {
		return nil, err
	}
	return VariableDeclarationsFromMaps(variables)
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

func SanitizeTemplatePipelineVariables(variables []map[string]any) []map[string]any {
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
		delete(copy, "stage_defaults")
		delete(copy, "stage_name")
		delete(copy, "stage_id")
		copy["name"], copy["source"], copy["editable"] = name, "pipeline_custom", true
		if _, exists := copy["secret"]; !exists {
			copy["secret"] = false
		}
		result = append(result, copy)
	}
	return result
}

func NormalizePipelineVariables(variables []map[string]any) ([]map[string]any, error) {
	result := make([]map[string]any, 0, len(variables))
	seen := make(map[string]struct{}, len(variables))
	for _, variable := range variables {
		name, _ := variable["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, apperror.New(apperror.KindValidation, "Pipeline variable name is required")
		}
		if !pipelineVariableNamePattern.MatchString(name) {
			return nil, apperror.New(apperror.KindValidation, "Invalid pipeline variable name: "+name)
		}
		if IsPipelineBuiltinVariable(name) {
			return nil, apperror.New(apperror.KindValidation, "Pipeline variable cannot override system context: "+name)
		}
		stageID, exists := variable["stage_id"].(string)
		if _, provided := variable["stage_id"]; provided && !exists {
			return nil, apperror.New(apperror.KindValidation, "Pipeline variable stage_id must be a string")
		}
		stageID = strings.TrimSpace(stageID)
		key := declarationScopeKey(name, stageID)
		if _, exists := seen[key]; exists {
			return nil, apperror.New(apperror.KindValidation, "Duplicate pipeline variable: "+name+" in scope "+stageID)
		}
		seen[key] = struct{}{}
		source, _ := variable["source"].(string)
		if source == "" {
			source = "pipeline_custom"
		}
		if source != "pipeline_custom" && source != "pipeline_stage" {
			return nil, apperror.New(apperror.KindValidation, "Invalid pipeline variable source: "+source)
		}
		copy := map[string]any{}
		maps.Copy(copy, variable)
		delete(copy, "stage_defaults")
		delete(copy, "stage_name")
		copy["name"], copy["source"], copy["editable"] = name, "pipeline_custom", true
		if stageID == "" {
			delete(copy, "stage_id")
		} else {
			copy["stage_id"] = stageID
		}
		if _, exists := copy["secret"]; !exists {
			copy["secret"] = false
		}
		result = append(result, copy)
	}
	return result, nil
}

// ValidatePipelineVariableScopes ensures every stage-scoped Pipeline variable
// references one of the Pipeline's current stage IDs.
func ValidatePipelineVariableScopes(variables []map[string]any, stages []model.StageDefinition) error {
	stageIDs := make(map[string]struct{}, len(stages))
	for _, stage := range stages {
		stageIDs[stageScopeID(stage)] = struct{}{}
	}
	for _, variable := range variables {
		stageID, _ := variable["stage_id"].(string)
		stageID = strings.TrimSpace(stageID)
		if stageID == "" {
			continue
		}
		if _, exists := stageIDs[stageID]; !exists {
			return apperror.New(apperror.KindValidation, "Pipeline variable stage_id does not exist: "+stageID)
		}
	}
	return nil
}

func ResolvePipelineVariablesFromStageDefinitions(stages []model.StageDefinition, custom []map[string]any) ([]map[string]any, error) {
	converted := make([]model.PipelineStage, 0, len(stages))
	for _, stage := range stages {
		artifacts, err := json.Marshal(stage.Artifacts)
		if err != nil {
			return nil, apperror.Wrap(apperror.KindInternal, "Invalid pipeline stage artifacts", err)
		}
		artifactData := string(artifacts)
		converted = append(converted, model.PipelineStage{Id: stage.Id, Name: stage.Name, Script: stage.Script, Artifacts: &artifactData})
	}
	return ResolvePipelineVariables(converted, custom)
}

func ResolvePipelineVariables(stages []model.PipelineStage, custom []map[string]any) ([]map[string]any, error) {
	extracted, err := extractPipelineStageDeclarationsFromPipelineStages(stages)
	if err != nil {
		return nil, err
	}
	custom, err = NormalizePipelineVariables(custom)
	if err != nil {
		return nil, err
	}
	stageDefinitions := make([]model.StageDefinition, 0, len(stages))
	for _, stage := range stages {
		stageDefinitions = append(stageDefinitions, model.StageDefinition{Id: stage.Id, Name: stage.Name})
	}
	if err := ValidatePipelineVariableScopes(custom, stageDefinitions); err != nil {
		return nil, err
	}
	customByKey := map[string]map[string]any{}
	for _, variable := range custom {
		name, _ := variable["name"].(string)
		stageID, _ := variable["stage_id"].(string)
		customByKey[declarationScopeKey(name, strings.TrimSpace(stageID))] = variable
	}
	result := make([]map[string]any, 0, len(extracted)+len(custom)+len(PipelineBuiltinVariableSpecs()))
	for _, name := range SortedPipelineBuiltinVariableNames() {
		result = append(result, PipelineBuiltinVariable(name))
	}
	consumed := map[string]struct{}{}
	for _, declaration := range extracted {
		key := declarationScopeKey(declaration.Name, declaration.StageId)
		if existing, ok := customByKey[key]; ok {
			copy := map[string]any{}
			maps.Copy(copy, existing)
			copy["stage_id"], copy["stage_name"], copy["stage_defaults"] = declaration.StageId, declaration.StageName, declaration.StageDefaults
			result = append(result, copy)
			consumed[key] = struct{}{}
			continue
		}
		result = append(result, declarationMap(declaration, false))
	}
	for _, variable := range custom {
		name, _ := variable["name"].(string)
		stageID, _ := variable["stage_id"].(string)
		key := declarationScopeKey(name, strings.TrimSpace(stageID))
		if _, exists := consumed[key]; !exists {
			result = append(result, variable)
		}
	}
	return result, nil
}

func ResolveTemplatePipelineVariables(stages []model.PipelineStage, custom []map[string]any) ([]map[string]any, error) {
	extracted, err := extractPipelineStageDeclarationsFromPipelineStages(stages)
	if err != nil {
		return nil, err
	}
	custom = SanitizeTemplatePipelineVariables(custom)
	customByKey := map[string]map[string]any{}
	for _, variable := range custom {
		name, _ := variable["name"].(string)
		stageID, _ := variable["stage_id"].(string)
		customByKey[declarationScopeKey(name, strings.TrimSpace(stageID))] = variable
	}
	result := make([]map[string]any, 0, len(extracted)+len(custom)+len(PipelineBuiltinVariableSpecs()))
	for _, name := range SortedPipelineBuiltinVariableNames() {
		result = append(result, PipelineBuiltinVariable(name))
	}
	consumed := map[string]struct{}{}
	for _, declaration := range extracted {
		key := declarationScopeKey(declaration.Name, declaration.StageId)
		if existing, ok := customByKey[key]; ok {
			copy := map[string]any{}
			maps.Copy(copy, existing)
			copy["stage_id"], copy["stage_name"], copy["stage_defaults"] = declaration.StageId, declaration.StageName, declaration.StageDefaults
			result = append(result, copy)
			consumed[key] = struct{}{}
			continue
		}
		declaration.Editable = false
		result = append(result, declarationMap(declaration, false))
	}
	for _, variable := range custom {
		name, _ := variable["name"].(string)
		stageID, _ := variable["stage_id"].(string)
		key := declarationScopeKey(name, strings.TrimSpace(stageID))
		if _, exists := consumed[key]; !exists {
			result = append(result, variable)
		}
	}
	return result, nil
}

func extractPipelineStageDeclarationsFromPipelineStages(stages []model.PipelineStage) ([]model.VariableDeclaration, error) {
	definitions := make([]model.StageDefinition, 0, len(stages))
	for _, stage := range stages {
		definition := model.StageDefinition{Id: stage.Id, Name: stage.Name, Script: stage.Script}
		if stage.Artifacts != nil && strings.TrimSpace(*stage.Artifacts) != "" {
			var artifacts []model.ArtifactConfig
			if err := json.Unmarshal([]byte(*stage.Artifacts), &artifacts); err != nil {
				return nil, apperror.New(apperror.KindValidation, "Invalid pipeline stage artifacts")
			}
			definition.Artifacts = artifacts
		}
		definitions = append(definitions, definition)
	}
	return extractStageVariableDeclarations(definitions)
}

func declarationMap(declaration model.VariableDeclaration, editable bool) map[string]any {
	data, _ := json.Marshal(declaration)
	result := map[string]any{}
	_ = json.Unmarshal(data, &result)
	if !editable {
		result["editable"] = false
	}
	if declaration.StageId == "" {
		delete(result, "stage_id")
	}
	if declaration.StageName == "" {
		delete(result, "stage_name")
	}
	return result
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
