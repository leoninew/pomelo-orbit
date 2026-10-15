package environmentsvc

import (
	"context"
	"testing"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type inventoryEnvironments struct {
	repository.EnvironmentStore
	items map[string]model.Environment
}

func (s inventoryEnvironments) EnvironmentByProject(_ context.Context, id string) (model.Environment, error) {
	item, ok := s.items[id]
	if !ok {
		return item, repository.ErrNotFound
	}
	return item, nil
}

func TestAccessInventoryDoesNotRequireCredentialsProbeOrActiveProject(t *testing.T) {
	projects := cliProjects{&terminalProjectReader{member: true}, []model.Project{{Id: "missing", Code: "z"}, {Id: "ssh", Code: "b", IsActive: false}, {Id: "local", Code: "a", IsActive: true}}}
	remote := readyTargetEnvironment()
	remote.SSH.CredentialId = ""
	remote.LastProbeStatus = nil
	service := New(inventoryEnvironments{items: map[string]model.Environment{"ssh": remote, "local": {Id: "local-env", TargetType: "local"}}}, projects, nil, "", nil, nil).WithLocalDisplay(environmentdto.LocalDisplaySnapshot{Platform: "linux", Host: "local-host", Username: "orbit"})
	access := NewAccessService(service, projects, NewTerminalService(service))
	items, err := access.List(context.Background(), "actor")
	if err != nil || len(items) != 3 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if items[0].Code != "a" || items[0].TargetType != "local" || items[0].Host != "local-host" || items[1].IsActive || items[1].TargetType != "ssh" || items[2].TargetType != "unconfigured" || items[2].EnvironmentId != "" {
		t.Fatalf("items=%+v", items)
	}
}

func TestCLITerminalSharesQuotaAndRequiresFinalSSHTarget(t *testing.T) {
	terminal, store, projects, _ := newTestTerminalService(t)
	access := NewAccessService(terminal.environments, cliProjects{projects, []model.Project{{Id: store.environment.ProjectId, Code: "production"}}}, terminal)
	var releases []func()
	for range 2 {
		grant, release, err := access.AcquireTerminal(context.Background(), "actor", "production")
		if err != nil || !grant.Target.Environment.IsSSH() {
			t.Fatal(err)
		}
		releases = append(releases, release)
		defer release()
	}
	if _, _, err := access.AcquireTerminal(context.Background(), "actor", "production"); err == nil {
		t.Fatal("CLI terminal bypassed quota")
	}
	releases[0]()
	grant, release, err := terminal.Acquire(context.Background(), "actor", store.environment.ProjectId)
	if err != nil || grant.ActorId != "actor" {
		t.Fatal(err)
	}
	release()
	store.environment.TargetType = "local"
	store.environment.SSH = nil
	if _, _, err := access.AcquireTerminal(context.Background(), "actor", "production"); err == nil {
		t.Fatal("CLI accepted local target")
	}
}
