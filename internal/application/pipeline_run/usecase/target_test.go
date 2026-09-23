package pipelinerunsvc

import (
	"context"
	"strings"
	"testing"
	"time"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
	pipelinerundto "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/dto"
	pipelinerunport "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type runTargetEnvironmentStore struct {
	repository.EnvironmentStore
	environment model.Environment
}

func (store *runTargetEnvironmentStore) EnvironmentByProject(context.Context, string) (model.Environment, error) {
	return store.environment, nil
}

type runTargetPipelineStore struct {
	directTriggerPipelineStore
	snapshot model.PipelineSnapshot
}

func (store *runTargetPipelineStore) LatestPipelineSnapshot(context.Context, string, string) (model.PipelineSnapshot, error) {
	store.snapshotRequested = true
	return store.snapshot, nil
}

func (store *runTargetPipelineStore) PipelineSnapshot(context.Context, string, string) (model.PipelineSnapshot, error) {
	return store.snapshot, nil
}

func (store *directTriggerPipelineRunStore) PipelineRunVersionBinding(context.Context, string, string) (model.PipelineRunVersionBinding, error) {
	return model.PipelineRunVersionBinding{}, repository.ErrNotFound
}

func readyLocalTarget() model.Environment {
	revision := int64(1)
	probeStatus := model.EnvironmentProbeStatusSucceeded
	return model.Environment{Id: "env-1", ProjectId: "project-1", TargetType: model.EnvironmentTargetTypeLocal,
		WorkspaceRoot: "/srv/orbit", TargetRevision: revision, LastProbeRevision: &revision, LastProbeStatus: &probeStatus}
}

func readySSHTarget() model.Environment {
	environment := readyLocalTarget()
	environment.TargetType = model.EnvironmentTargetTypeSSH
	environment.SSH = &model.EnvironmentSSHTarget{Platform: model.EnvironmentPlatformWindows, Host: "example.test", Port: 22,
		Username: "orbit", CredentialId: "credential-1", CredentialRevision: 1,
		HostKeyFingerprint: "SHA256:abcdefghijklmnopqrstuvwxyz0123456789abcde="}
	return environment
}

func targetTestService(environment *runTargetEnvironmentStore, repo model.Repository) (Service, *directTriggerPipelineRunStore) {
	projectId, repositoryId := "project-1", "repo-1"
	pipeline := model.Pipeline{Id: "pipeline-1", ProjectId: &projectId, RepositoryId: &repositoryId,
		Kind: model.PipelineKindApplication, Version: 1, VariableDeclarations: `[]`}
	snapshot := model.PipelineSnapshot{Id: "snapshot-1", PipelineVersion: pipeline.Version, StagesSnapshot: `[]`}
	pipelineStore := &runTargetPipelineStore{directTriggerPipelineStore: directTriggerPipelineStore{pipeline: pipeline}, snapshot: snapshot}
	runStore := &directTriggerPipelineRunStore{}
	service := Service{store: stores{project: directTriggerProjectStore{}, repository: directTriggerRepositoryStore{repository: repo},
		pipeline: pipelineStore, pipelineRun: runStore}, environments: environment,
		targetResolver: environmentsvc.NewTargetResolver(environment, nil, ""),
		workspace:      directTriggerWorkspace{}}
	return service, runStore
}

type directTriggerWorkspace struct{ pipelinerunport.Workspace }

func (directTriggerWorkspace) WorkspaceForTarget(context.Context, model.Environment) (pipelinerunport.Workspace, error) {
	return directTriggerWorkspace{}, nil
}

func TestTriggerAndRetryRequireFreshProjectProbe(t *testing.T) {
	for _, action := range []string{"trigger", "retry"} {
		for _, scenario := range []string{"stale local", "failed local", "unbound ssh"} {
			t.Run(action+"/"+scenario, func(t *testing.T) {
				environment := readyLocalTarget()
				switch scenario {
				case "stale local":
					environment.TargetRevision++
				case "failed local":
					probeStatus := model.EnvironmentProbeStatusFailed
					environment.LastProbeStatus = &probeStatus
				case "unbound ssh":
					environment = readySSHTarget()
				}
				store := &runTargetEnvironmentStore{environment: environment}
				repo := model.Repository{Id: "repo-1", Code: "repo", RepositoryType: model.RepositoryTypeRemoteGit,
					RepositoryUrl: "https://git.example.test/repo.git", DefaultBranch: "main"}
				service, runStore := targetTestService(store, repo)
				var err error
				if action == "retry" {
					runStore.run = model.PipelineRun{Id: "previous", PipelineId: "pipeline-1", Status: status.WorkStatusFaulted,
						VariablesSnapshot: `[]`}
					_, err = service.RetryPipelineRun(context.Background(), "user-1", "project-1", "previous")
				} else {
					_, err = service.TriggerPipeline(context.Background(), "user-1", "project-1", "pipeline-1", "")
				}
				if err == nil || !apperror.IsKind(err, apperror.KindValidation) || runStore.createdRun != nil {
					t.Fatalf("%s error=%v, created=%v", action, err, runStore.createdRun)
				}
			})
		}
	}
}

func TestSSHTriggerRejectsUnusableRepositoryBeforeSnapshot(t *testing.T) {
	for _, testCase := range []struct{ name, repositoryType, url, want string }{
		{"local directory", model.RepositoryTypeLocalDirectory, "/srv/source", "Local directory"},
		{"git ssh", model.RepositoryTypeRemoteGit, "git@example.test:org/repo.git", "HTTPS Git"},
		{"git http", model.RepositoryTypeRemoteGit, "http://example.test/repo.git", "HTTPS Git"},
		{"malformed https", model.RepositoryTypeRemoteGit, "https:///repo.git", "HTTPS Git"},
		{"valid https runner unavailable", model.RepositoryTypeRemoteGit, "https://example.test/repo.git", "not installed"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			store := &runTargetEnvironmentStore{environment: readySSHTarget()}
			repo := model.Repository{Id: "repo-1", Code: "repo", RepositoryType: testCase.repositoryType,
				RepositoryUrl: testCase.url, DefaultBranch: "main"}
			service, runStore := targetTestService(store, repo)
			service.targetResolver = directTriggerTargetResolver{environment: store.environment}
			_, err := service.TriggerPipeline(context.Background(), "user-1", "project-1", "pipeline-1", "")
			if err == nil || !strings.Contains(err.Error(), testCase.want) || runStore.createdRun != nil ||
				service.store.pipeline.(*runTargetPipelineStore).snapshotRequested {
				t.Fatalf("error=%v, created=%v", err, runStore.createdRun)
			}
		})
	}
}

func TestRunTargetSnapshotAndRetryBindsCurrentEnvironment(t *testing.T) {
	environment := &runTargetEnvironmentStore{environment: readyLocalTarget()}
	repo := model.Repository{Id: "repo-1", Code: "repo", RepositoryType: model.RepositoryTypeLocalDirectory,
		RepositoryUrl: "/srv/source", DefaultBranch: "main"}
	service, runStore := targetTestService(environment, repo)
	first, err := service.TriggerPipeline(context.Background(), "user-1", "project-1", "pipeline-1", "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Run.EnvironmentId == nil || *first.Run.EnvironmentId != environment.environment.Id ||
		first.Run.EnvironmentTargetRevision == nil || *first.Run.EnvironmentTargetRevision != 1 || first.Run.SSHCredentialId != nil {
		t.Fatalf("first run target: %+v", first.Run)
	}
	runStore.run = first.Run
	environment.environment.TargetRevision = 2
	environment.environment.LastProbeRevision = &environment.environment.TargetRevision
	second, err := service.RetryPipelineRun(context.Background(), "user-1", "project-1", first.Run.Id)
	if err != nil {
		t.Fatal(err)
	}
	if second.Run.RetryOf == nil || *second.Run.RetryOf != first.Run.Id ||
		second.Run.EnvironmentTargetRevision == nil || *second.Run.EnvironmentTargetRevision != 2 {
		t.Fatalf("retry target: %+v", second.Run)
	}
}

func TestQueuedRunRejectsChangedTargetBeforeLocalSideEffects(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		mutate func(*model.Environment)
	}{
		{"environment revision", func(environment *model.Environment) { environment.TargetRevision++ }},
		{"environment identity", func(environment *model.Environment) { environment.Id = "replacement" }},
		{"credential revision", func(environment *model.Environment) { environment.SSH.CredentialRevision++ }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			environment := readySSHTarget()
			run := model.PipelineRun{Id: "run-1", CreatedAt: time.Now()}
			applyRunTargetSnapshot(&run, structTarget(environment))
			ssh := *environment.SSH
			environment.SSH = &ssh
			testCase.mutate(&environment)
			store := &runTargetEnvironmentStore{environment: environment}
			service := Service{environments: store, targetResolver: directTriggerTargetResolver{environment: environment}}
			_, err := service.resolveQueuedTarget(context.Background(), "project-1", run)
			if err == nil || !strings.Contains(err.Error(), "retry run") {
				t.Fatalf("changed target error = %v", err)
			}
		})
	}
}

func structTarget(environment model.Environment) environmentport.Target {
	return environmentport.Target{Environment: environment}
}

func TestHistoricalRunOnlyReadsUnchangedLocalTarget(t *testing.T) {
	now := time.Now()
	environment := readyLocalTarget()
	environment.CreatedAt = now.Add(-time.Hour)
	environment.UpdatedAt = now.Add(-time.Minute)
	store := &runTargetEnvironmentStore{environment: environment}
	service := Service{environments: store, workspace: directTriggerWorkspace{}}
	run := model.PipelineRun{CreatedAt: now, Id: "historical"}
	if _, err := service.workspaceForRun(context.Background(), "project-1", run); err != nil {
		t.Fatal(err)
	}
	store.environment.UpdatedAt = now.Add(-time.Millisecond)
	if _, err := service.workspaceForRun(context.Background(), "project-1", run); err == nil {
		t.Fatal("same-second timestamps cannot prove a historical local target is unchanged")
	}
	store.environment.UpdatedAt = now.Add(time.Second)
	if _, err := service.workspaceForRun(context.Background(), "project-1", run); err == nil {
		t.Fatal("changed local environment cannot locate historical run")
	}
	store.environment = readySSHTarget()
	if _, err := service.workspaceForRun(context.Background(), "project-1", run); err == nil {
		t.Fatal("historical run must never read remote or control-plane files after SSH switch")
	}
}

func TestSSHRunNeverSelectsControlPlaneResources(t *testing.T) {
	environment := readySSHTarget()
	service := Service{environments: &runTargetEnvironmentStore{environment: environment}, workspace: directTriggerWorkspace{}}
	run := model.PipelineRun{Id: "run-1"}
	applyRunTargetSnapshot(&run, structTarget(environment))
	_, err := service.workspaceForRun(context.Background(), "project-1", run)
	if err == nil || !strings.Contains(err.Error(), "remote pipeline workspace") {
		t.Fatalf("SSH workspace error = %v", err)
	}
}

type queuedRunStore struct {
	pipelineExecutionStore
	run              model.PipelineRun
	loadedRepository bool
	completedStatus  string
	completedMessage string
}

func (store *queuedRunStore) BeginPipelineRun(context.Context, string, string) (bool, error) {
	return true, nil
}

func (store *queuedRunStore) PipelineRun(context.Context, string, string) (model.PipelineRun, error) {
	return store.run, nil
}

func (store *queuedRunStore) Repository(context.Context, string, string) (model.Repository, error) {
	store.loadedRepository = true
	return model.Repository{Id: "repo-1", RepositoryType: model.RepositoryTypeRemoteGit}, nil
}

func (store *queuedRunStore) CompletePipelineRun(_ context.Context, _, _ string, statusValue, message string) (bool, error) {
	store.completedStatus, store.completedMessage = statusValue, message
	return true, nil
}

func TestQueuedRunFailsBeforeRepositoryAccessOnTargetChange(t *testing.T) {
	environment := readyLocalTarget()
	run := model.PipelineRun{Id: "run-1", RepositoryId: "repo-1"}
	applyRunTargetSnapshot(&run, structTarget(environment))
	environment.TargetRevision++
	store := &queuedRunStore{run: run}
	service := Service{environments: &runTargetEnvironmentStore{environment: environment},
		targetResolver: directTriggerTargetResolver{environment: environment}, executionStore: store,
		workspace: directTriggerWorkspace{}}
	if err := service.ExecutePipelineRun(context.Background(), pipelinerundto.ExecutePipelineRunInput{
		ProjectId: "project-1", PipelineRunId: run.Id,
	}); err != nil {
		t.Fatal(err)
	}
	if store.loadedRepository || store.completedStatus != status.WorkStatusFaulted ||
		!strings.Contains(store.completedMessage, "retry run") {
		t.Fatalf("queued run: repository accessed=%v status=%q message=%q",
			store.loadedRepository, store.completedStatus, store.completedMessage)
	}
}

func TestQueuedSSHRunCannotFallBackToLocalRunner(t *testing.T) {
	environment := readySSHTarget()
	run := model.PipelineRun{Id: "run-1", RepositoryId: "repo-1"}
	applyRunTargetSnapshot(&run, structTarget(environment))
	store := &queuedRunStore{run: run}
	service := Service{environments: &runTargetEnvironmentStore{environment: environment},
		targetResolver: directTriggerTargetResolver{environment: environment}, executionStore: store,
		workspace: directTriggerWorkspace{}}
	if err := service.ExecutePipelineRun(context.Background(), pipelinerundto.ExecutePipelineRunInput{
		ProjectId: "project-1", PipelineRunId: run.Id,
	}); err != nil {
		t.Fatal(err)
	}
	if store.completedStatus != status.WorkStatusFaulted ||
		!strings.Contains(store.completedMessage, "not installed") {
		t.Fatalf("SSH run status=%q message=%q", store.completedStatus, store.completedMessage)
	}
}

type runTargetLogStore struct {
	*directTriggerPipelineRunStore
	stageRun model.PipelineStageRun
}

func (store runTargetLogStore) PipelineStageRun(context.Context, string, string) (model.PipelineStageRun, error) {
	return store.stageRun, nil
}

func TestChangedRunTargetCannotReadControlPlaneStageLog(t *testing.T) {
	environment := readyLocalTarget()
	run := model.PipelineRun{Id: "run-1"}
	applyRunTargetSnapshot(&run, structTarget(environment))
	environment.TargetRevision++
	store := runTargetLogStore{directTriggerPipelineRunStore: &directTriggerPipelineRunStore{run: run},
		stageRun: model.PipelineStageRun{Id: "stage-run-1", PipelineRunId: run.Id}}
	service := Service{store: stores{project: directTriggerProjectStore{}, pipelineRun: store},
		environments: &runTargetEnvironmentStore{environment: environment}, workspace: directTriggerWorkspace{}}
	_, err := service.PipelineStageLog(context.Background(), "user-1", "project-1", run.Id, store.stageRun.Id, 0)
	if err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("log read after target change error = %v", err)
	}
}
