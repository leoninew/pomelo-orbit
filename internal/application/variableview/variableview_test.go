package variableview

import (
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestSnapshotUsesThreeVariableKinds(t *testing.T) {
	variables, err := Snapshot(nil, []model.VariableDeclaration{
		{Name: "runtime_datetime"},
		{Name: "repository_code"},
		{Name: "image_name"},
	})
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	kinds := make(map[string]Kind, len(variables))
	for _, variable := range variables {
		kinds[variable.Name] = variable.Kind
	}
	for name, want := range map[string]Kind{
		"runtime_datetime": KindSystemGenerated,
		"repository_code":  KindRepositoryContext,
		"image_name":       KindPipelineContext,
	} {
		if got := kinds[name]; got != want {
			t.Errorf("kind for %s = %q, want %q", name, got, want)
		}
	}
}

func TestPipelineUsesStageLiquidDefaultAsConfigurationDefault(t *testing.T) {
	variables, err := Pipeline(model.Pipeline{Kind: model.PipelineKindApplication, VariableDeclarations: "[]"}, []model.StageDefinition{{
		Id:     "stage-build",
		Name:   "docker build",
		Script: `cd {{ working_dir | default: "." }}`,
	}}, nil)
	if err != nil {
		t.Fatalf("Pipeline() error = %v", err)
	}

	for _, variable := range variables {
		if variable.Name == "working_dir" && variable.StageBinding != nil && variable.StageBinding.StageId == "stage-build" {
			if variable.Configuration == nil || variable.Configuration.Default != "." || variable.Configuration.Editable {
				t.Fatalf("working_dir configuration = %#v", variable.Configuration)
			}
			return
		}
	}
	t.Fatal("working_dir stage variable was not found")
}

func TestSnapshotDisplaysStageDefaultWithoutChangingRunValue(t *testing.T) {
	declarations := []model.VariableDeclaration{{
		Name: "working_dir", StageId: "stage-build", StageName: "docker build",
		Source: "pipeline_stage", StageDefaults: []model.StageVariableDefault{
			{StageId: "stage-build", StageName: "docker build", Default: "."},
			{StageId: "stage-build", StageName: "docker build", Default: "."},
		},
	}}
	variables, err := Snapshot(nil, declarations)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if len(variables) != 1 || variables[0].Configuration == nil || variables[0].Configuration.Default != "." || variables[0].Configuration.Value != nil {
		t.Fatalf("snapshot variable = %#v", variables)
	}
	if declarations[0].Value != nil || declarations[0].Default != nil {
		t.Fatalf("snapshot declaration was modified: %#v", declarations[0])
	}
}

func TestSnapshotStageValueTakesPriorityOverExpressionDefault(t *testing.T) {
	variables, err := Snapshot(nil, []model.VariableDeclaration{{
		Name: "working_dir", StageId: "stage-build", Value: "backend",
		StageDefaults: []model.StageVariableDefault{{Default: "."}},
	}})
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if len(variables) != 1 || variables[0].Configuration == nil || variables[0].Configuration.Value != "backend" || variables[0].Configuration.Default != nil {
		t.Fatalf("snapshot variable = %#v", variables)
	}
}

func TestSnapshotDoesNotPresentConflictingExpressionDefaultsAsOneValue(t *testing.T) {
	variables, err := Snapshot(nil, []model.VariableDeclaration{{
		Name: "working_dir", StageId: "stage-build",
		StageDefaults: []model.StageVariableDefault{{Default: "web"}, {Default: "webapi"}},
	}})
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if len(variables) != 1 || variables[0].Configuration == nil || variables[0].Configuration.Default != nil {
		t.Fatalf("snapshot variable = %#v", variables)
	}
}
