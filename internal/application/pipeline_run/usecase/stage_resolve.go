package pipelinerunsvc

import (
	"fmt"

	pipelinevariable "github.com/leoninew/pomelo-orbit/internal/application/pipeline/rule/pipelinevariable"
	templatex "github.com/leoninew/pomelo-orbit/internal/common/template"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func resolveStages(stages []model.StageDefinition, runtime pipelinevariable.RuntimeVariables) ([]model.StageDefinition, error) {
	resolved := make([]model.StageDefinition, len(stages))
	copy(resolved, stages)
	for i := range resolved {
		values := runtime.ValuesForStage(resolved[i])
		script, err := renderStageField(resolved[i], "script", resolved[i].Script, values)
		if err != nil {
			return nil, err
		}
		resolved[i].Script = script
		for j := range resolved[i].Artifacts {
			reference, err := renderStageField(resolved[i], fmt.Sprintf("artifacts[%d].reference", j), resolved[i].Artifacts[j].Reference, values)
			if err != nil {
				return nil, err
			}
			name, err := renderStageField(resolved[i], fmt.Sprintf("artifacts[%d].name", j), resolved[i].Artifacts[j].Name, values)
			if err != nil {
				return nil, err
			}
			command, err := renderStageField(resolved[i], fmt.Sprintf("artifacts[%d].command", j), resolved[i].Artifacts[j].Command, values)
			if err != nil {
				return nil, err
			}
			resolved[i].Artifacts[j].Reference = reference
			resolved[i].Artifacts[j].Name = name
			resolved[i].Artifacts[j].Command = command
		}
	}
	return resolved, nil
}

func renderStageField(stage model.StageDefinition, field, text string, variables map[string]any) (string, error) {
	if _, err := templatex.ExtractPipelineVariableReferences(text); err != nil {
		return "", fmt.Errorf("stage %s %s: %w", stage.Name, field, err)
	}
	value, err := templatex.Render(text, variables)
	if err != nil {
		return "", fmt.Errorf("stage %s %s: %w", stage.Name, field, err)
	}
	return value, nil
}
