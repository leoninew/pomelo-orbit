package status

import "testing"

func TestTaskTypeValues(t *testing.T) {
	values := map[string]string{
		"pipeline run":       TaskTypePipelineRunExecute,
		"deployment deploy":  TaskTypeDeploymentDeploy,
		"deployment restart": TaskTypeDeploymentRestart,
		"deployment stop":    TaskTypeDeploymentStop,
	}
	want := map[string]string{
		"pipeline run":       "pipeline_run.execute",
		"deployment deploy":  "deployment.deploy",
		"deployment restart": "deployment.restart",
		"deployment stop":    "deployment.stop",
	}
	for name, value := range values {
		if value != want[name] {
			t.Fatalf("%s task type = %q, want %q", name, value, want[name])
		}
	}
}
