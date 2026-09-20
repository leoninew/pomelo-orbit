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

	pipelineVariableKindRepository      = "repository_variable"
	pipelineVariableKindPipeline        = "pipeline_variable"
	pipelineVariableKindStage           = "stage_variable"
	pipelineVariableKindRuntimeInput    = "runtime_input"
	pipelineVariableKindSystemContext   = "system_context"
	pipelineVariableKindSystemGenerated = "system_generated"

	pipelineVariableValueSourceRepository    = "repository_variable"
	pipelineVariableValueSourcePipeline      = "pipeline_variable"
	pipelineVariableValueSourceStageOverride = "stage_override"
	pipelineVariableValueSourceLiquidDefault = "liquid_default"
	pipelineVariableValueSourceRuntime       = "runtime_input"
	pipelineVariableValueSourceSystem        = "system_generated"
	pipelineVariableValueSourceSnapshot      = "snapshot"
	pipelineVariableValueSourceMissing       = "missing"
)

// View is the shared, derived variable read model used by resource details.
// It is never persisted as Pipeline or Repository configuration.
type View struct {
	Name                string
	Kind                string
	Scope               string
	StageBinding        *StageBinding
	References          []Reference
	Configuration       *Configuration
	GlobalConfiguration *Configuration
	StageOverride       *Configuration
	ValueSource         string
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

type Reference struct {
	StageId       string
	StageName     string
	Field         string
	ArtifactName  string
	ArtifactIndex *int
	Default       any
	HasDefault    bool
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
	if pipeline.Kind == model.PipelineKindTemplate {
		pipelineVariables = pipelinevariable.SanitizeTemplatePipelineVariables(pipelineVariables)
	} else {
		pipelineVariables, err = pipelinevariable.NormalizePipelineVariables(pipelineVariables)
		if err != nil {
			return nil, err
		}
		if err := pipelinevariable.ValidatePipelineVariableScopes(pipelineVariables, stages); err != nil {
			return nil, err
		}
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
	stageReferences := make(map[string][]pipelinevariable.StageVariableReference)
	globalReferences := make(map[string][]pipelinevariable.StageVariableReference)
	for _, stage := range stages {
		stageNames[pipelinevariable.StageScopeId(stage)] = stage.Name
	}
	for _, reference := range references {
		globalReferences[reference.Name] = append(globalReferences[reference.Name], reference)
		if !pipelinevariable.IsPipelineBuiltinVariable(reference.Name) {
			key := declarationScopeKey(reference.Name, reference.StageId)
			stageReferences[key] = append(stageReferences[key], reference)
		}
	}

	result := make([]View, 0, len(repositoryVariables)+len(pipelineGlobals)+len(stageReferences)+len(pipelineStages)+len(pipelinevariable.PipelineBuiltinVariableSpecs()))
	for _, name := range sortedVariableDeclarationNames(repositoryVariables) {
		declaration := repositoryVariables[name]
		result = append(result, View{
			Name:          name,
			Kind:          pipelineVariableKindRepository,
			Scope:         pipelineVariableScopeGlobal,
			References:    variableReferenceViews(globalReferences[name]),
			Configuration: variableConfiguration(declaration, false),
			ValueSource:   pipelineVariableValueSourceRepository,
		})
	}
	for _, name := range sortedVariableDeclarationNames(pipelineGlobals) {
		declaration := pipelineGlobals[name]
		result = append(result, View{
			Name:          name,
			Kind:          pipelineVariableKindPipeline,
			Scope:         pipelineVariableScopeGlobal,
			References:    variableReferenceViews(globalReferences[name]),
			Configuration: variableConfiguration(declaration, true),
			ValueSource:   globalValueSource(name, repositoryVariables),
			Editable:      true,
		})
	}
	result = append(result, builtinVariableViews(repo, globalReferences)...)

	for _, key := range sortedStageReferenceKeys(stageReferences) {
		references := stageReferences[key]
		first := references[0]
		stageOverride, hasStageOverride := pipelineStages[key]
		globalConfiguration, hasGlobalConfiguration := pipelineGlobals[first.Name]
		variable := View{
			Name:         first.Name,
			Kind:         pipelineVariableKindStage,
			Scope:        pipelineVariableScopeStage,
			StageBinding: &StageBinding{StageId: first.StageId, StageName: first.StageName},
			References:   variableReferenceViews(references),
			ValueSource:  stageValueSource(first.Name, first.StageId, repositoryVariables, pipelineGlobals, pipelineStages, references),
			Editable:     pipeline.Kind != model.PipelineKindTemplate,
		}
		if hasGlobalConfiguration {
			variable.GlobalConfiguration = variableConfiguration(globalConfiguration, true)
		}
		if hasStageOverride {
			variable.StageOverride = variableConfiguration(stageOverride, true)
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
			Kind:          pipelineVariableKindPipeline,
			Scope:         pipelineVariableScopeStage,
			StageBinding:  &StageBinding{StageId: declaration.StageId, StageName: stageName},
			Configuration: variableConfiguration(declaration, true),
			ValueSource:   stageValueSource(declaration.Name, declaration.StageId, repositoryVariables, pipelineGlobals, pipelineStages, nil),
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
		result = append(result, View{Name: declaration.Name, Kind: pipelineVariableKindRepository, Scope: pipelineVariableScopeGlobal, References: []Reference{}, Configuration: variableConfiguration(declaration, true), ValueSource: pipelineVariableValueSourceRepository, Editable: true})
	}
	result = append(result,
		View{Name: "repository_code", Kind: pipelineVariableKindSystemContext, Scope: pipelineVariableScopeGlobal, References: []Reference{}, Configuration: &Configuration{Description: pipelinevariable.PipelineBuiltinVariableSpecs()["repository_code"], Value: repo.Code}, ValueSource: pipelineVariableValueSourceSystem},
		View{Name: "repository_url", Kind: pipelineVariableKindSystemContext, Scope: pipelineVariableScopeGlobal, References: []Reference{}, Configuration: &Configuration{Description: pipelinevariable.PipelineBuiltinVariableSpecs()["repository_url"], Value: repo.RepositoryUrl}, ValueSource: pipelineVariableValueSourceSystem},
		View{Name: "repository_ref", Kind: pipelineVariableKindRuntimeInput, Scope: pipelineVariableScopeGlobal, References: []Reference{}, Configuration: &Configuration{Description: pipelinevariable.PipelineBuiltinVariableSpecs()["repository_ref"], Default: repo.DefaultBranch}, ValueSource: pipelineVariableValueSourceRuntime},
	)
	sortVariableViews(result)
	return result, nil
}

func Snapshot(stages []model.StageDefinition, declarations []model.VariableDeclaration) ([]View, error) {
	references, err := pipelinevariable.ExtractStageVariableReferences(stages)
	if err != nil {
		return nil, err
	}
	globalReferences := make(map[string][]pipelinevariable.StageVariableReference)
	stageReferences := make(map[string][]pipelinevariable.StageVariableReference)
	for _, reference := range references {
		globalReferences[reference.Name] = append(globalReferences[reference.Name], reference)
		stageReferences[declarationScopeKey(reference.Name, reference.StageId)] = append(stageReferences[declarationScopeKey(reference.Name, reference.StageId)], reference)
	}
	result := make([]View, 0, len(declarations))
	for _, declaration := range declarations {
		scope := pipelineVariableScopeGlobal
		var binding *StageBinding
		variableReferences := globalReferences[declaration.Name]
		if declaration.StageId != "" {
			scope = pipelineVariableScopeStage
			binding = &StageBinding{StageId: declaration.StageId, StageName: declaration.StageName}
			variableReferences = stageReferences[declarationScopeKey(declaration.Name, declaration.StageId)]
		}
		result = append(result, View{Name: declaration.Name, Kind: snapshotVariableKind(declaration, len(variableReferences) > 0), Scope: scope, StageBinding: binding, References: variableReferenceViews(variableReferences), Configuration: variableConfiguration(declaration, false), ValueSource: pipelineVariableValueSourceSnapshot})
	}
	sortVariableViews(result)
	return result, nil
}

func snapshotVariableKind(declaration model.VariableDeclaration, hasReferences bool) string {
	switch declaration.Name {
	case "repository_ref":
		return pipelineVariableKindRuntimeInput
	case "repository_code", "repository_url":
		return pipelineVariableKindSystemContext
	case "runtime_datetime":
		return pipelineVariableKindSystemGenerated
	}
	if declaration.Source == "repository" {
		return pipelineVariableKindRepository
	}
	if declaration.StageId != "" && hasReferences {
		return pipelineVariableKindStage
	}
	return pipelineVariableKindPipeline
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

func builtinVariableViews(repo *model.Repository, references map[string][]pipelinevariable.StageVariableReference) []View {
	result := make([]View, 0, len(pipelinevariable.PipelineBuiltinVariableSpecs()))
	for _, name := range pipelinevariable.SortedPipelineBuiltinVariableNames() {
		variable := View{
			Name:       name,
			Scope:      pipelineVariableScopeGlobal,
			References: variableReferenceViews(references[name]),
		}
		switch name {
		case "repository_ref":
			variable.Kind = pipelineVariableKindRuntimeInput
			variable.ValueSource = pipelineVariableValueSourceRuntime
			configuration := Configuration{Description: pipelinevariable.PipelineBuiltinVariableSpecs()[name], Editable: false}
			if repo != nil {
				configuration.Default = repo.DefaultBranch
			}
			variable.Configuration = &configuration
		case "runtime_datetime":
			variable.Kind = pipelineVariableKindSystemGenerated
			variable.ValueSource = pipelineVariableValueSourceSystem
			variable.Configuration = &Configuration{Description: pipelinevariable.PipelineBuiltinVariableSpecs()[name], Editable: false}
		default:
			variable.Kind = pipelineVariableKindSystemContext
			variable.ValueSource = pipelineVariableValueSourceSystem
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

func globalValueSource(name string, repository map[string]model.VariableDeclaration) string {
	if declaration, exists := repository[name]; exists {
		if _, hasValue := pipelinevariable.EffectiveVariableValue(declaration); hasValue {
			return pipelineVariableValueSourceRepository
		}
	}
	return pipelineVariableValueSourcePipeline
}

func stageValueSource(name, stageId string, repository, globals, stages map[string]model.VariableDeclaration, references []pipelinevariable.StageVariableReference) string {
	if declaration, exists := repository[name]; exists {
		if _, hasValue := pipelinevariable.EffectiveVariableValue(declaration); hasValue {
			return pipelineVariableValueSourceRepository
		}
	}
	if declaration, exists := stages[declarationScopeKey(name, stageId)]; exists {
		if _, hasValue := pipelinevariable.EffectiveVariableValue(declaration); hasValue {
			return pipelineVariableValueSourceStageOverride
		}
	}
	if declaration, exists := globals[name]; exists {
		if _, hasValue := pipelinevariable.EffectiveVariableValue(declaration); hasValue {
			return pipelineVariableValueSourcePipeline
		}
	}
	for _, reference := range references {
		if reference.HasDefault {
			return pipelineVariableValueSourceLiquidDefault
		}
	}
	return pipelineVariableValueSourceMissing
}

func variableConfiguration(declaration model.VariableDeclaration, editable bool) *Configuration {
	return &Configuration{Description: declaration.Description, Default: declaration.Default, Value: declaration.Value, Secret: declaration.Secret, Editable: editable}
}

func variableReferenceViews(references []pipelinevariable.StageVariableReference) []Reference {
	result := make([]Reference, 0, len(references))
	for _, reference := range references {
		result = append(result, Reference{StageId: reference.StageId, StageName: reference.StageName, Field: reference.Field, ArtifactName: reference.ArtifactName, ArtifactIndex: reference.ArtifactIndex, Default: reference.Default, HasDefault: reference.HasDefault})
	}
	return result
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

func sortedStageReferenceKeys(values map[string][]pipelinevariable.StageVariableReference) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := values[result[i]][0], values[result[j]][0]
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
