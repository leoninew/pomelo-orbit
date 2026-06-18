package cisvc

import (
	"encoding/json"
	"strings"
	"time"

	"backend/internal/apperror"
	"backend/internal/repository/model"
)

const maskedSecretValue = "***"

func buildPipelineRunVariables(repo model.Repository, template model.PipelineTemplate, snapshot model.PipelineSnapshot, triggerRef string, runtimeOverrides map[string]string) (string, error) {
	declarations, err := completeSnapshotVariableDeclarations(snapshot, template)
	if err != nil {
		return "", err
	}
	variables, err := buildRuntimeVariables(repo, template, triggerRef, runtimeOverrides, declarations)
	if err != nil {
		return "", err
	}
	if err := validateRuntimeVariables(variables, declarations); err != nil {
		return "", err
	}
	return marshalRuntimeVariableSnapshot(variables, declarations)
}

func completeSnapshotVariableDeclarations(snapshot model.PipelineSnapshot, template model.PipelineTemplate) ([]model.VariableDeclaration, error) {
	variables, err := variableDeclarations(snapshot.VariablesSnapshot)
	if err != nil {
		return nil, err
	}
	if len(variables) > 0 {
		return variables, nil
	}
	stages, err := pipelineSnapshotStages(snapshot.StagesSnapshot)
	if err != nil {
		return nil, err
	}
	custom, err := pipelineTemplateVariables(template.VariableDeclarations)
	if err != nil {
		return nil, err
	}
	resolved := resolveTemplateVariablesFromStageDefinitions(stages, custom)
	return variableDeclarationsFromMaps(resolved)
}

func buildRuntimeVariables(repo model.Repository, template model.PipelineTemplate, triggerRef string, runtimeOverrides map[string]string, declarations []model.VariableDeclaration) (map[string]any, error) {
	variables := map[string]any{}
	for name, value := range repositoryBuiltinVariables(repo, triggerRef) {
		variables[name] = value
	}
	for name, value := range templateBuiltinVariables(template) {
		variables[name] = value
	}
	for name, value := range runtimeOverrides {
		name = strings.TrimSpace(name)
		if name == "" || isPipelineTemplateBuiltinVariable(name) {
			continue
		}
		variables[name] = value
	}
	repoVariables, err := variableDeclarations(repo.VariableOverrides)
	if err != nil {
		return nil, err
	}
	for _, declaration := range repoVariables {
		if _, exists := variables[declaration.Name]; !exists && declaration.Value != nil && !isPipelineTemplateBuiltinVariable(declaration.Name) {
			variables[declaration.Name] = declaration.Value
		}
	}
	templateVariables, err := variableDeclarations(template.VariableDeclarations)
	if err != nil {
		return nil, err
	}
	for _, declaration := range templateVariables {
		if _, exists := variables[declaration.Name]; !exists && !isPipelineTemplateBuiltinVariable(declaration.Name) {
			if value, ok := effectiveVariableValue(declaration); ok {
				variables[declaration.Name] = value
			}
		}
	}
	for _, declaration := range declarations {
		if _, exists := variables[declaration.Name]; exists || isPipelineTemplateBuiltinVariable(declaration.Name) {
			continue
		}
		if value, ok := effectiveVariableValue(declaration); ok {
			variables[declaration.Name] = value
		}
	}
	return variables, nil
}

func repositoryBuiltinVariables(repo model.Repository, triggerRef string) map[string]any {
	return map[string]any{
		"repository_id":   repo.Id,
		"repository_name": repo.Name,
		"repository_code": repo.Code,
		"repository_url":  repo.RepositoryURL,
		"repository_ref":  triggerRef,
	}
}

func templateBuiltinVariables(template model.PipelineTemplate) map[string]any {
	return map[string]any{
		"template_id":      template.Id,
		"template_name":    template.Name,
		"template_version": template.Version,
		"runtime_datetime": time.Now().UTC().Format("20060102-150405"),
	}
}

func effectiveVariableValue(declaration model.VariableDeclaration) (any, bool) {
	if hasRuntimeValue(declaration.Value) {
		return declaration.Value, true
	}
	if hasRuntimeValue(declaration.Default) {
		return declaration.Default, true
	}
	return nil, false
}

func validateRuntimeVariables(variables map[string]any, declarations []model.VariableDeclaration) error {
	missing := make([]string, 0)
	for _, declaration := range declarations {
		if isPipelineTemplateBuiltinVariable(declaration.Name) {
			continue
		}
		if !hasRuntimeValue(variables[declaration.Name]) {
			missing = append(missing, declaration.Name)
		}
	}
	if len(missing) > 0 {
		return apperror.New(apperror.KindValidation, "Missing variable value: "+strings.Join(missing, ", "))
	}
	return nil
}

func marshalRuntimeVariableSnapshot(variables map[string]any, declarations []model.VariableDeclaration) (string, error) {
	snapshot := make([]model.VariableDeclaration, 0, len(declarations))
	for _, declaration := range declarations {
		value := variables[declaration.Name]
		if declaration.Secret && hasRuntimeValue(value) {
			value = maskedSecretValue
		}
		declaration.Value = value
		snapshot = append(snapshot, declaration)
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return "", apperror.Wrap(apperror.KindValidation, "Invalid variables", err)
	}
	return string(data), nil
}

func variableDeclarations(value string) ([]model.VariableDeclaration, error) {
	if strings.TrimSpace(value) == "" {
		return []model.VariableDeclaration{}, nil
	}
	var declarations []model.VariableDeclaration
	if err := json.Unmarshal([]byte(value), &declarations); err == nil {
		return declarations, nil
	}
	var legacy map[string]any
	if err := json.Unmarshal([]byte(value), &legacy); err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Invalid variable declarations", err)
	}
	declarations = make([]model.VariableDeclaration, 0, len(legacy))
	for name, value := range legacy {
		declarations = append(declarations, model.VariableDeclaration{Name: name, Value: value, Source: "runtime", Editable: true})
	}
	return declarations, nil
}

func variableDeclarationsFromMaps(values []map[string]any) ([]model.VariableDeclaration, error) {
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

func resolveTemplateVariablesFromStageDefinitions(stages []model.StageDefinition, custom []map[string]any) []map[string]any {
	converted := make([]model.BuildStage, 0, len(stages))
	for _, stage := range stages {
		artifacts, _ := json.Marshal(stage.Artifacts)
		artifactString := string(artifacts)
		converted = append(converted, model.BuildStage{Name: stage.Name, Script: stage.Script, Artifacts: &artifactString})
	}
	return resolveTemplateVariables(converted, custom)
}

func hasRuntimeValue(value any) bool {
	if value == nil {
		return false
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) != ""
	}
	return true
}
