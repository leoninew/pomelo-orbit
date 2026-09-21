package variableview

import (
	"sort"
	"strings"

	pipelinevariable "github.com/leoninew/pomelo-orbit/internal/application/pipeline/rule/pipelinevariable"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

const (
	pipelineVariableScopeGlobal = "global"
	pipelineVariableScopeStage  = "stage"
)

type Kind string

const (
	KindSystemGenerated   Kind = "system_generated"
	KindRepositoryContext Kind = "repository_context"
	KindPipelineContext   Kind = "pipeline_context"
)

// View is the shared, derived variable read model used by resource details.
// It is never persisted as Pipeline or Repository configuration.
type View struct {
	Name                string
	Kind                Kind
	Scope               string
	StageBinding        *StageBinding
	Configuration       *Configuration
	GlobalConfiguration *Configuration
	StageOverride       *Configuration
	Editable            bool
}

type Configuration struct {
	Description string
	Default     any
	Value       any
	Secret      bool
	Editable    bool
}

type StageBinding struct {
	StageId   string
	StageName string
}

func declarationScopeKey(name, stageId string) string {
	return name + "\x00" + stageId
}

// Pipeline constructs the configuration view used by a Pipeline detail. It
// deliberately does not resolve a concrete Run value.
func Pipeline(pipeline model.Pipeline, stages []model.StageDefinition, repo *model.Repository) ([]View, error) {
	pipelineVariables, err := pipelinevariable.PipelineVariables(pipeline.VariableDeclarations)
	if err != nil {
		return nil, err
	}
	pipelineVariables, err = pipelinevariable.NormalizePipelineVariables(pipelineVariables)
	if err != nil {
		return nil, err
	}
	if err := pipelinevariable.ValidatePipelineVariableScopes(pipelineVariables, stages); err != nil {
		return nil, err
	}

	pipelineGlobals, pipelineStages, err := pipelineVariableConfigurations(pipelineVariables)
	if err != nil {
		return nil, err
	}
	repositoryVariables, err := pipelineRepositoryVariables(repo)
	if err != nil {
		return nil, err
	}
	references, err := pipelinevariable.ExtractStageVariableReferences(stages)
	if err != nil {
		return nil, err
	}

	stageNames := make(map[string]string, len(stages))
	stageReferences := make(map[string]pipelinevariable.StageVariableReference)
	for _, stage := range stages {
		stageNames[pipelinevariable.StageScopeId(stage)] = stage.Name
	}
	for _, reference := range references {
		if pipelinevariable.IsPipelineBuiltinVariable(reference.Name) {
			continue
		}
		key := declarationScopeKey(reference.Name, reference.StageId)
		if _, exists := stageReferences[key]; !exists {
			stageReferences[key] = reference
		}
	}

	result := make([]View, 0, len(repositoryVariables)+len(pipelineGlobals)+len(stageReferences)+len(pipelineStages)+len(pipelinevariable.PipelineBuiltinVariableSpecs()))
	for _, name := range sortedVariableDeclarationNames(repositoryVariables) {
		declaration := repositoryVariables[name]
		result = append(result, View{
			Name:          name,
			Kind:          KindRepositoryContext,
			Scope:         pipelineVariableScopeGlobal,
			Configuration: variableConfiguration(declaration, false),
		})
	}
	for _, name := range sortedVariableDeclarationNames(pipelineGlobals) {
		declaration := pipelineGlobals[name]
		result = append(result, View{
			Name:          name,
			Kind:          KindPipelineContext,
			Scope:         pipelineVariableScopeGlobal,
			Configuration: variableConfiguration(declaration, true),
			Editable:      true,
		})
	}
	result = append(result, builtinVariableViews(repo)...)

	for _, key := range sortedStageReferenceKeys(stageReferences) {
		first := stageReferences[key]
		stageOverride, hasStageOverride := pipelineStages[key]
		globalConfiguration, hasGlobalConfiguration := pipelineGlobals[first.Name]
		variable := View{
			Name:         first.Name,
			Kind:         KindPipelineContext,
			Scope:        pipelineVariableScopeStage,
			StageBinding: &StageBinding{StageId: first.StageId, StageName: first.StageName},
			Editable:     true,
		}
		if hasGlobalConfiguration {
			variable.GlobalConfiguration = variableConfiguration(globalConfiguration, true)
		}
		if hasStageOverride {
			variable.StageOverride = variableConfiguration(stageOverride, true)
		} else if !hasGlobalConfiguration && first.HasDefault {
			variable.Configuration = &Configuration{Default: first.Default}
		}
		result = append(result, variable)
	}

	for _, key := range sortedVariableDeclarationKeys(pipelineStages) {
		if _, exists := stageReferences[key]; exists {
			continue
		}
		declaration := pipelineStages[key]
		stageName := stageNames[declaration.StageId]
		result = append(result, View{
			Name:          declaration.Name,
			Kind:          KindPipelineContext,
			Scope:         pipelineVariableScopeStage,
			StageBinding:  &StageBinding{StageId: declaration.StageId, StageName: stageName},
			Configuration: variableConfiguration(declaration, true),
			Editable:      true,
		})
	}

	sortVariableViews(result)
	return result, nil
}

func PipelineStages(pipeline model.Pipeline, stages []model.PipelineStage, repo *model.Repository) ([]View, error) {
	definitions, err := pipelinevariable.StageDefinitionsFromPipelineStages(stages)
	if err != nil {
		return nil, err
	}
	return Pipeline(pipeline, definitions, repo)
}

func Repository(repo model.Repository) ([]View, error) {
	declarations, err := pipelinevariable.RepositoryVariableDeclarations(repo.VariableOverrides)
	if err != nil {
		return nil, err
	}
	result := make([]View, 0, len(declarations)+3)
	for _, declaration := range declarations {
		result = append(result, View{Name: declaration.Name, Kind: KindRepositoryContext, Scope: pipelineVariableScopeGlobal, Configuration: variableConfiguration(declaration, true), Editable: true})
	}
	result = append(result,
		View{Name: "repository_code", Kind: KindRepositoryContext, Scope: pipelineVariableScopeGlobal, Configuration: &Configuration{Description: pipelinevariable.PipelineBuiltinVariableSpecs()["repository_code"], Value: repo.Code}},
		View{Name: "repository_url", Kind: KindRepositoryContext, Scope: pipelineVariableScopeGlobal, Configuration: &Configuration{Description: pipelinevariable.PipelineBuiltinVariableSpecs()["repository_url"], Value: repo.RepositoryUrl}},
		View{Name: "repository_ref", Kind: KindRepositoryContext, Scope: pipelineVariableScopeGlobal, Configuration: &Configuration{Description: pipelinevariable.PipelineBuiltinVariableSpecs()["repository_ref"], Default: repo.DefaultBranch}},
	)
	sortVariableViews(result)
	return result, nil
}

func Snapshot(_ []model.StageDefinition, declarations []model.VariableDeclaration) ([]View, error) {
	result := make([]View, 0, len(declarations))
	for _, declaration := range declarations {
		scope := pipelineVariableScopeGlobal
		var binding *StageBinding
		if declaration.StageId != "" {
			scope = pipelineVariableScopeStage
			binding = &StageBinding{StageId: declaration.StageId, StageName: declaration.StageName}
		}
		result = append(result, View{Name: declaration.Name, Kind: snapshotVariableKind(declaration), Scope: scope, StageBinding: binding, Configuration: variableConfiguration(declaration, false)})
	}
	sortVariableViews(result)
	return result, nil
}

func snapshotVariableKind(declaration model.VariableDeclaration) Kind {
	if declaration.Name == "runtime_datetime" {
		return KindSystemGenerated
	}
	if strings.HasPrefix(declaration.Name, "repository_") || declaration.Source == "repository" {
		return KindRepositoryContext
	}
	return KindPipelineContext
}

func sortVariableViews(variables []View) {
	sort.Slice(variables, func(i, j int) bool {
		if variables[i].Scope != variables[j].Scope {
			return variables[i].Scope < variables[j].Scope
		}
		if variables[i].StageBinding != nil || variables[j].StageBinding != nil {
			left, right := "", ""
			if variables[i].StageBinding != nil {
				left = variables[i].StageBinding.StageName + "\x00" + variables[i].StageBinding.StageId
			}
			if variables[j].StageBinding != nil {
				right = variables[j].StageBinding.StageName + "\x00" + variables[j].StageBinding.StageId
			}
			if left != right {
				return left < right
			}
		}
		if variables[i].Name != variables[j].Name {
			return variables[i].Name < variables[j].Name
		}
		return variables[i].Kind < variables[j].Kind
	})
}

func pipelineVariableConfigurations(variables []map[string]any) (map[string]model.VariableDeclaration, map[string]model.VariableDeclaration, error) {
	globals := make(map[string]model.VariableDeclaration)
	stages := make(map[string]model.VariableDeclaration)
	for _, raw := range variables {
		declaration, err := pipelinevariable.VariableDeclarationFromMap(raw, "pipeline")
		if err != nil {
			return nil, nil, err
		}
		if declaration.StageId == "" {
			globals[declaration.Name] = declaration
			continue
		}
		stages[declarationScopeKey(declaration.Name, declaration.StageId)] = declaration
	}
	return globals, stages, nil
}

func pipelineRepositoryVariables(repo *model.Repository) (map[string]model.VariableDeclaration, error) {
	result := make(map[string]model.VariableDeclaration)
	if repo == nil {
		return result, nil
	}
	declarations, err := pipelinevariable.RepositoryVariableDeclarations(repo.VariableOverrides)
	if err != nil {
		return nil, err
	}
	for _, declaration := range declarations {
		result[declaration.Name] = declaration
	}
	return result, nil
}

func builtinVariableViews(repo *model.Repository) []View {
	result := make([]View, 0, len(pipelinevariable.PipelineBuiltinVariableSpecs()))
	for _, name := range pipelinevariable.SortedPipelineBuiltinVariableNames() {
		variable := View{
			Name:  name,
			Scope: pipelineVariableScopeGlobal,
		}
		switch name {
		case "repository_ref":
			variable.Kind = KindRepositoryContext
			configuration := Configuration{Description: pipelinevariable.PipelineBuiltinVariableSpecs()[name], Editable: false}
			if repo != nil {
				configuration.Default = repo.DefaultBranch
			}
			variable.Configuration = &configuration
		case "runtime_datetime":
			variable.Kind = KindSystemGenerated
			variable.Configuration = &Configuration{Description: pipelinevariable.PipelineBuiltinVariableSpecs()[name], Editable: false}
		default:
			variable.Kind = KindRepositoryContext
			configuration := Configuration{Description: pipelinevariable.PipelineBuiltinVariableSpecs()[name], Editable: false}
			if repo != nil {
				configuration.Value = pipelinevariable.SystemVariableValue(*repo, name)
			}
			variable.Configuration = &configuration
		}
		result = append(result, variable)
	}
	return result
}

func variableConfiguration(declaration model.VariableDeclaration, editable bool) *Configuration {
	return &Configuration{Description: declaration.Description, Default: declaration.Default, Value: declaration.Value, Secret: declaration.Secret, Editable: editable}
}

func sortedVariableDeclarationNames(values map[string]model.VariableDeclaration) []string {
	result := make([]string, 0, len(values))
	for name := range values {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func sortedVariableDeclarationKeys(values map[string]model.VariableDeclaration) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := values[result[i]], values[result[j]]
		if left.StageId != right.StageId {
			return left.StageId < right.StageId
		}
		return left.Name < right.Name
	})
	return result
}

func sortedStageReferenceKeys(values map[string]pipelinevariable.StageVariableReference) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := values[result[i]], values[result[j]]
		if left.StageName != right.StageName {
			return left.StageName < right.StageName
		}
		if left.StageId != right.StageId {
			return left.StageId < right.StageId
		}
		return strings.Compare(left.Name, right.Name) < 0
	})
	return result
}
