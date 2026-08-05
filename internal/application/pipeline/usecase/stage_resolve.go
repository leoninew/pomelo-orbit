package pipelinesvc

import (
	templatex "gitee.com/leoninew/PomeloOrbit-go/internal/common/template"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func resolveStages(stages []model.StageDefinition, variables map[string]any) ([]model.StageDefinition, error) {
	resolved := make([]model.StageDefinition, len(stages))
	copy(resolved, stages)
	for i := range resolved {
		script, err := templatex.Render(resolved[i].Script, variables)
		if err != nil {
			return nil, err
		}
		resolved[i].Script = script
		for j := range resolved[i].Artifacts {
			reference, err := templatex.Render(resolved[i].Artifacts[j].Reference, variables)
			if err != nil {
				return nil, err
			}
			name, err := templatex.Render(resolved[i].Artifacts[j].Name, variables)
			if err != nil {
				return nil, err
			}
			command, err := templatex.Render(resolved[i].Artifacts[j].Command, variables)
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
