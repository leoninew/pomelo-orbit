package pipelinerunsvc

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
			path, err := templatex.Render(resolved[i].Artifacts[j].Path, variables)
			if err != nil {
				return nil, err
			}
			name, err := templatex.Render(resolved[i].Artifacts[j].Name, variables)
			if err != nil {
				return nil, err
			}
			resolved[i].Artifacts[j].Path = path
			resolved[i].Artifacts[j].Name = name
		}
	}
	return resolved, nil
}
