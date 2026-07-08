package cisvc

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

var templateVariablePattern = regexp.MustCompile(`\{\{\s*([A-Za-z][A-Za-z0-9_]*)(?:\s*\|\s*default\s*:\s*['\"]([^'\"]*)['\"])?\s*\}\}`)

type ArtifactConfig struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Name string `json:"name"`
}

func pipelineSnapshotStages(value string) ([]model.StageDefinition, error) {
	if strings.TrimSpace(value) == "" {
		return []model.StageDefinition{}, nil
	}
	var stages []model.StageDefinition
	if err := json.Unmarshal([]byte(value), &stages); err != nil {
		return nil, apperror.New(apperror.KindInternal, "Invalid pipeline snapshot stages")
	}
	return stages, nil
}

func pipelineTemplateVariables(value string) ([]map[string]any, error) {
	if strings.TrimSpace(value) == "" {
		return []map[string]any{}, nil
	}
	var variables []map[string]any
	if err := json.Unmarshal([]byte(value), &variables); err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Invalid pipeline template variables", err)
	}
	return variables, nil
}

func sanitizePipelineTemplateVariables(variables []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(variables))
	for _, variable := range variables {
		name, _ := variable["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" || isPipelineTemplateBuiltinVariable(name) {
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
		for key, value := range variable {
			copy[key] = value
		}
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

func resolveTemplateVariables(stages []model.BuildStage, custom []map[string]any) []map[string]any {
	extracted := map[string]any{}
	for _, stage := range stages {
		extractTemplateVariables(stage.Script, extracted)
		if stage.Artifacts != nil && strings.TrimSpace(*stage.Artifacts) != "" {
			var artifacts []ArtifactConfig
			if err := json.Unmarshal([]byte(*stage.Artifacts), &artifacts); err == nil {
				for _, artifact := range artifacts {
					extractTemplateVariables(artifact.Path, extracted)
					extractTemplateVariables(artifact.Name, extracted)
				}
			}
		}
	}
	custom = sanitizePipelineTemplateVariables(custom)
	customByName := make(map[string]map[string]any, len(custom))
	for _, variable := range custom {
		name, _ := variable["name"].(string)
		customByName[name] = variable
	}
	builtinNames := sortedPipelineTemplateBuiltinVariableNames()
	result := []map[string]any{}
	if len(extracted) == 0 {
		for _, name := range builtinNames {
			result = append(result, pipelineTemplateBuiltinVariable(name))
		}
		for _, variable := range custom {
			name, _ := variable["name"].(string)
			if !isPipelineTemplateBuiltinVariable(name) {
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
		if isPipelineTemplateBuiltinVariable(name) {
			result = append(result, pipelineTemplateBuiltinVariable(name))
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

func extractTemplateVariables(text string, found map[string]any) {
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

func sortedPipelineTemplateBuiltinVariableNames() []string {
	names := make([]string, 0, len(pipelineTemplateBuiltinVariableSpecs()))
	for name := range pipelineTemplateBuiltinVariableSpecs() {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func pipelineTemplateBuiltinVariable(name string) map[string]any {
	spec := pipelineTemplateBuiltinVariableSpecs()[name]
	return map[string]any{"name": name, "description": spec, "default": nil, "value": nil, "secret": false, "source": "template", "editable": false}
}
