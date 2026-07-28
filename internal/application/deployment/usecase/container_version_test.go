package deploymentsvc

import (
	"testing"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
)

func TestParseAndApplyContainerVersionLabels(t *testing.T) {
	t.Parallel()

	labelsByID, err := parseContainerLabelOutput("full-container-id {\"pomelo.orbit.version-id\":\"version-1\",\"pomelo.orbit.version-label\":\"2026.07.28\"}")
	if err != nil {
		t.Fatal(err)
	}
	containers := []deploymentdto.RuntimeContainer{{ID: "full-cont", Name: "demo-web-1"}}
	applyContainerVersionLabels(containers, labelsByID)
	if got, want := containers[0].VersionID, "version-1"; got != want {
		t.Fatalf("VersionID: got %q want %q", got, want)
	}
	if got, want := containers[0].VersionLabel, "2026.07.28"; got != want {
		t.Fatalf("VersionLabel: got %q want %q", got, want)
	}
}

func TestParseContainerLabelOutputRejectsInvalidRecord(t *testing.T) {
	t.Parallel()

	if _, err := parseContainerLabelOutput("missing-json"); err == nil {
		t.Fatal("expected invalid record error")
	}
}
