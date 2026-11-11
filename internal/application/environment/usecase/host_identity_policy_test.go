package environmentsvc

import (
	"context"
	"reflect"
	"testing"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestHostIdentityPolicyLifecycleAcrossSaveEntrypoints(t *testing.T) {
	for _, entrypoint := range []string{"initialization", "update", "prepare"} {
		t.Run(entrypoint, func(t *testing.T) {
			for _, scenario := range []struct {
				name     string
				before   string
				incoming *string
				changed  bool
			}{
				{name: "omitted", before: model.EnvironmentHostIdentityPolicyStable},
				{name: "same", before: model.EnvironmentHostIdentityPolicyStable, incoming: policyPointer(model.EnvironmentHostIdentityPolicyStable)},
				{name: "enable", before: model.EnvironmentHostIdentityPolicyNegotiated, incoming: policyPointer(model.EnvironmentHostIdentityPolicyStable), changed: true},
				{name: "disable", before: model.EnvironmentHostIdentityPolicyStable, incoming: policyPointer(model.EnvironmentHostIdentityPolicyNegotiated), changed: true},
			} {
				t.Run(scenario.name, func(t *testing.T) {
					before := testProbeEnvironment("project-1")
					before.SSH.HostIdentityPolicy = scenario.before
					revision, status := before.TargetRevision, model.EnvironmentProbeStatusSucceeded
					before.LastProbeRevision, before.LastProbeStatus = &revision, &status
					credentials := credentialsForEnvironment(t, before, "private-key")
					originalCredential := credentials.items[0]
					store := &initializationEnvironmentStore{environment: before, credentials: credentials, found: true}
					prober := &probeEnvironmentProber{}
					service := New(store, initializationProjectReader{}, credentials, testCredentialSecret, prober, nil)
					input := environmentdto.UpdateInput{
						TargetType: &before.TargetType,
						SSH: &environmentdto.SSHTargetInput{
							Platform: before.SSH.Platform, Host: before.SSH.Host, Port: before.SSH.Port,
							Username: before.SSH.Username, WorkspaceRoot: before.WorkspaceRoot, HostIdentityPolicy: scenario.incoming,
						},
					}
					var view environmentdto.View
					var err error
					switch entrypoint {
					case "initialization":
						view, err = service.SaveInitialization(context.Background(), "user", before.ProjectId, input)
					case "update":
						input.TargetType = nil
						view, err = service.UpdateForUser(context.Background(), "user", before.ProjectId, input)
					case "prepare":
						var command environmentdto.SSHCommandView
						command, err = service.PrepareSSHEnvironment(context.Background(), "user", before.ProjectId, input)
						view = command.Environment
					}
					if err != nil {
						t.Fatal(err)
					}
					expectedPolicy, expectedPin, expectedRevision := scenario.before, before.SSH.HostKeyFingerprint, before.TargetRevision
					if scenario.incoming != nil {
						expectedPolicy = *scenario.incoming
					}
					if scenario.changed {
						expectedPin, expectedRevision = "", expectedRevision+1
					}
					if view.SSH.HostIdentityPolicy != expectedPolicy || view.SSH.HostKeyFingerprint != expectedPin || view.TargetRevision != expectedRevision {
						t.Fatalf("policy lifecycle: view=%+v ssh=%+v", view, view.SSH)
					}
					if view.LastProbeRevision == nil || *view.LastProbeRevision != revision || view.LastProbeStatus == nil || *view.LastProbeStatus != status {
						t.Fatalf("save rewrote probe history: %+v", view)
					}
					if prober.called || len(credentials.items) != 1 || !reflect.DeepEqual(credentials.items[0], originalCredential) {
						t.Fatal("save connected to SSH or changed deployment credentials")
					}
				})
			}
		})
	}
}

func TestHostIdentityPolicyRejectsInvalidExternalValuesBeforeSaving(t *testing.T) {
	for _, policy := range []string{"", "unknown"} {
		t.Run(policy, func(t *testing.T) {
			before := testProbeEnvironment("project-1")
			store := &updateEnvironmentStore{environment: before}
			_, err := New(store, probeProjectReader{}, nil, "", nil, nil).
				UpdateForUser(context.Background(), "user", before.ProjectId, environmentdto.UpdateInput{
					SSH: &environmentdto.SSHTargetInput{
						Platform: before.SSH.Platform, Host: before.SSH.Host, Port: before.SSH.Port,
						Username: before.SSH.Username, WorkspaceRoot: before.WorkspaceRoot, HostIdentityPolicy: &policy,
					},
				})
			if !apperror.IsKind(err, apperror.KindValidation) || store.updated {
				t.Fatalf("invalid policy: err=%v saved=%v", err, store.updated)
			}
		})
	}
}

func policyPointer(policy string) *string {
	return &policy
}
