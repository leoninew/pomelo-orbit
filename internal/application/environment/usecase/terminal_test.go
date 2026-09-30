package environmentsvc

import (
	"context"
	"testing"
	"time"

	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type terminalProjectReader struct {
	repository.ProjectReader
	member bool
}

func (p *terminalProjectReader) Project(_ context.Context, id string) (model.Project, error) {
	return model.Project{Id: id}, nil
}

func (p *terminalProjectReader) IsProjectMember(context.Context, string, string) (bool, error) {
	return p.member, nil
}

func newTestTerminalService(t *testing.T) (*TerminalService, *targetEnvironmentStore, *terminalProjectReader, *memoryEnvironmentCredentials) {
	t.Helper()
	store := &targetEnvironmentStore{environment: readyTargetEnvironment()}
	projects := &terminalProjectReader{member: true}
	credentials := credentialsForEnvironment(t, store.environment, "managed-private-key")
	environments := New(store, projects, credentials, testCredentialSecret, nil, nil)
	return NewTerminalService(environments, NewTargetResolver(store, credentials, testCredentialSecret)), store, projects, credentials
}

func TestTerminalTicketBindsManagedIdentityAndCannotBeReplayed(t *testing.T) {
	svc, store, _, _ := newTestTerminalService(t)
	ctx := context.Background()
	ticket, err := svc.IssueTicket(ctx, "orbit-user", store.environment.ProjectId)
	if err != nil {
		t.Fatal(err)
	}
	grant, release, err := svc.RedeemTicket(ctx, store.environment.ProjectId, ticket)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if grant.ActorId != "orbit-user" || grant.Target.Environment.SSH.Username != "orbit" ||
		grant.Target.Environment.SSH.Host != store.environment.SSH.Host || grant.Target.PrivateKey.PrivateKey != "managed-private-key" {
		t.Fatal("terminal did not bind the actor and managed SSH identity")
	}
	if _, _, err := svc.RedeemTicket(ctx, store.environment.ProjectId, ticket); !apperror.IsKind(err, apperror.KindUnauthorized) {
		t.Fatalf("replayed ticket error = %v", err)
	}
}

func TestTerminalRejectsUnavailableEnvironmentAndNonMember(t *testing.T) {
	for _, scenario := range []string{"local", "stale probe", "non-member"} {
		t.Run(scenario, func(t *testing.T) {
			svc, store, projects, _ := newTestTerminalService(t)
			switch scenario {
			case "local":
				store.environment.TargetType = model.EnvironmentTargetTypeLocal
				store.environment.SSH = nil
			case "stale probe":
				store.environment.TargetRevision++
			case "non-member":
				projects.member = false
			}
			if _, err := svc.IssueTicket(context.Background(), "actor", store.environment.ProjectId); err == nil {
				t.Fatal("issued terminal ticket for an unavailable environment or non-member")
			}
		})
	}
}

func TestTerminalTicketExpiresAndPinsTargetRevision(t *testing.T) {
	for _, scenario := range []string{"expired", "target changed", "credential changed", "membership removed"} {
		t.Run(scenario, func(t *testing.T) {
			svc, store, projects, credentials := newTestTerminalService(t)
			ticket, err := svc.IssueTicket(context.Background(), "actor", store.environment.ProjectId)
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "expired":
				for key, value := range svc.tickets {
					value.expiresAt = time.Now().Add(-time.Second)
					svc.tickets[key] = value
				}
			case "target changed":
				store.environment.TargetRevision++
				store.environment.LastProbeRevision = &store.environment.TargetRevision
			case "credential changed":
				credentials.items[0].Revision++
			case "membership removed":
				projects.member = false
			}
			if _, _, err := svc.RedeemTicket(context.Background(), store.environment.ProjectId, ticket); err == nil {
				t.Fatal("redeemed an expired or stale terminal ticket")
			}
		})
	}
}

func TestTerminalQuotaIsReleasedOnce(t *testing.T) {
	svc, store, _, _ := newTestTerminalService(t)
	ctx := context.Background()
	issue := func() string {
		t.Helper()
		ticket, err := svc.IssueTicket(ctx, "actor", store.environment.ProjectId)
		if err != nil {
			t.Fatal(err)
		}
		return ticket
	}
	var releases []func()
	for range maxUserTerminals {
		_, release, err := svc.RedeemTicket(ctx, store.environment.ProjectId, issue())
		if err != nil {
			t.Fatal(err)
		}
		releases = append(releases, release)
		defer release()
	}
	if _, _, err := svc.RedeemTicket(ctx, store.environment.ProjectId, issue()); !apperror.IsKind(err, apperror.KindUnavailable) {
		t.Fatalf("over-quota ticket error = %v", err)
	}
	releases[0]()
	releases[0]()
	_, release, err := svc.RedeemTicket(ctx, store.environment.ProjectId, issue())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
}

func TestTerminalSessionRejectsRevokedMembershipTargetOrCredential(t *testing.T) {
	for _, scenario := range []string{"membership", "target", "credential"} {
		t.Run(scenario, func(t *testing.T) {
			svc, store, projects, credentials := newTestTerminalService(t)
			ctx := context.Background()
			ticket, err := svc.IssueTicket(ctx, "actor", store.environment.ProjectId)
			if err != nil {
				t.Fatal(err)
			}
			grant, release, err := svc.RedeemTicket(ctx, store.environment.ProjectId, ticket)
			if err != nil {
				t.Fatal(err)
			}
			defer release()
			if err := svc.ValidateSession(ctx, grant); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "membership":
				projects.member = false
			case "target":
				store.environment.TargetRevision++
			case "credential":
				credentials.items[0].Revision++
			}
			if err := svc.ValidateSession(ctx, grant); err == nil {
				t.Fatal("continued terminal session after access or target changed")
			}
		})
	}
}
