package pipelinerunsvc

import (
	pipelinevariable "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/rule/pipelinevariable"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func buildPipelineRunVariables(repo model.Repository, template model.PipelineTemplate, snapshot model.PipelineSnapshot, triggerRef string, runtimeOverrides map[string]string) (string, error) {
	declarations, err := pipelinevariable.CompleteSnapshotVariableDeclarations(snapshot, template)
	if err != nil {
		return "", err
	}
	variables, err := pipelinevariable.BuildRuntimeVariables(repo, template, triggerRef, runtimeOverrides, declarations)
	if err != nil {
		return "", err
	}
	if err := pipelinevariable.ValidateRuntimeVariables(variables, declarations); err != nil {
		return "", err
	}
	return pipelinevariable.MarshalRuntimeVariableSnapshot(variables, declarations)
}
