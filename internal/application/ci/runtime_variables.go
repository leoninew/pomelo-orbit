package cisvc

import (
	"gitee.com/leoninew/PomeloOrbit-go/internal/common/civariable"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func buildPipelineRunVariables(repo model.Repository, template model.PipelineTemplate, snapshot model.PipelineSnapshot, triggerRef string, runtimeOverrides map[string]string) (string, error) {
	declarations, err := civariable.CompleteSnapshotVariableDeclarations(snapshot, template)
	if err != nil {
		return "", err
	}
	variables, err := civariable.BuildRuntimeVariables(repo, template, triggerRef, runtimeOverrides, declarations)
	if err != nil {
		return "", err
	}
	if err := civariable.ValidateRuntimeVariables(variables, declarations); err != nil {
		return "", err
	}
	return civariable.MarshalRuntimeVariableSnapshot(variables, declarations)
}
