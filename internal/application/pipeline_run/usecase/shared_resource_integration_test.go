package pipelinerunsvc

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	credentialsvc "github.com/leoninew/pomelo-orbit/internal/application/credential/usecase"
	pipelinedto "github.com/leoninew/pomelo-orbit/internal/application/pipeline/dto"
	pipelinesvc "github.com/leoninew/pomelo-orbit/internal/application/pipeline/usecase"
	pipelinerundto "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/dto"
	repositorydto "github.com/leoninew/pomelo-orbit/internal/application/repository/dto"
	repositorysvc "github.com/leoninew/pomelo-orbit/internal/application/repository/usecase"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/config"
	databasepkg "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	applicationrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/application"
	credentialrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/credential"
	pipelinerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/pipeline"
	pipelinerunrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/pipeline_run"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
	vcsrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/repository"
)

func TestSharedRepositoryRunsInEachProjectWithSharedCredential(t *testing.T) {
	const (
		userId     = "01KKX2YNPF6VJ9N7QYCWG61KVK"
		projectA   = "01KRRKK0K3T519ZQZES3M4QA9Z"
		projectB   = "shared-run-project"
		secretKey  = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
		repoURL    = "https://git.example.test/shared.git"
		secretData = "shared-token"
	)
	ctx := context.Background()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	database.SetMaxOpenConns(1)
	if err := databasepkg.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO project (id, name, code) VALUES (?, ?, ?)`, projectB, "Shared run", "shared-run"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO project_member (project_id, user_id) VALUES (?, ?)`, projectB, userId); err != nil {
		t.Fatal(err)
	}

	projects := projectrepo.NewRepository(database)
	credentials := credentialrepo.NewRepository(database)
	repositories := vcsrepo.NewRepository(database)
	pipelines := pipelinerepo.NewRepository(database)
	runs := pipelinerunrepo.NewRepository(database)
	applications := applicationrepo.NewRepository(database)
	credentialService := credentialsvc.New(projects, credentials, repositories, secretKey)
	repositoryService := repositorysvc.New(projects, credentials, repositories, nil, nil)
	pipelineService := pipelinesvc.New(projects, pipelines, runs, applications, repositories, nil)
	credential, err := credentialService.CreateCredential(ctx, userId, credentialdto.CredentialCreateInput{
		ProjectId: projectA, Name: "Shared run token", Type: "github_token", Data: secretData,
	})
	if err != nil {
		t.Fatal(err)
	}
	repo, err := repositoryService.CreateRepository(ctx, userId, repositorydto.RepositoryCreateInput{
		ProjectId: projectA, Name: "Shared run source", Code: "shared-run-source",
		RepositoryType: model.RepositoryTypeRemoteGit, RepositoryUrl: repoURL,
		GitCredentialId: &credential.Id, DefaultBranch: "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	stageVersion := 1
	artifacts := "[]"
	if err := pipelines.CreatePipelineStageTemplate(ctx, model.PipelineStage{
		Id: "shared-run-stage", Kind: model.PipelineStageKindTemplate, Name: "Clone", Image: "alpine/git", Description: "Clone shared source",
		Script: "git clone {{ repository_url }}", Artifacts: &artifacts, Version: &stageVersion,
	}); err != nil {
		t.Fatal(err)
	}
	template := model.Pipeline{
		Id: "shared-run-template", Kind: model.PipelineKindTemplate, Name: "Shared clone",
		VariableDeclarations: "[]", Version: 1,
	}
	if err := pipelines.CreatePipeline(ctx, template); err != nil {
		t.Fatal(err)
	}
	if err := pipelines.UpdateTemplatePipelineWithReferences(ctx, projectA, template, []model.PipelineStageReference{{
		Id: "shared-run-reference", PipelineId: template.Id, SourceTemplateStageId: "shared-run-stage",
		SourceTemplateStageName: "Clone", SourceTemplateStageVersion: 1, Name: "Clone",
		SourceTemplateStageDescription: "Clone shared source",
		Image:                          "alpine/git", Script: "git clone {{ repository_url }}", Artifacts: "[]", DependsOn: "[]",
	}}); err != nil {
		t.Fatal(err)
	}

	runIds := make(map[string]string)
	for _, projectId := range []string{projectA, projectB} {
		instance, err := pipelineService.InstantiatePipeline(ctx, userId, projectId, template.Id, pipelinedto.PipelineInstantiateInput{
			Name: "Shared clone " + projectId, RepositoryId: repo.Repository.Id,
		})
		if err != nil {
			t.Fatalf("instantiate in %s: %v", projectId, err)
		}
		environment := readySSHTarget()
		environment.Id = "env-" + projectId
		environment.ProjectId = projectId
		runner := &remoteRunnerStub{}
		service := NewExecutionService(projects, credentials, repositories, pipelines, runs, applications,
			&runTargetEnvironmentStore{environment: environment}, directTriggerTargetResolver{environment: environment},
			nil, nil, directTriggerWorkspace{}, secretKey, slog.New(slog.NewTextHandler(io.Discard, nil)),
			30*time.Second, time.Second, nil, nil, nil,
		).WithRemoteRuntime(remoteResourcesStub{workspace: &remoteWorkspaceStub{}, runner: runner, logs: &remoteLogStub{}})
		created, err := service.TriggerPipeline(ctx, userId, projectId, instance.Pipeline.Id, "main")
		if err != nil {
			t.Fatalf("trigger in %s: %v", projectId, err)
		}
		if err := service.ExecutePipelineRun(ctx, pipelinerundto.ExecutePipelineRunInput{ProjectId: projectId, PipelineRunId: created.Run.Id}); err != nil {
			t.Fatalf("execute in %s: %v", projectId, err)
		}
		stored, err := runs.PipelineRun(ctx, projectId, created.Run.Id)
		if err != nil {
			t.Fatal(err)
		}
		if stored.Status != status.WorkStatusRanToCompletion || stored.ProjectId == nil || *stored.ProjectId != projectId ||
			stored.RepositoryId != repo.Repository.Id || stored.EnvironmentId == nil || *stored.EnvironmentId != environment.Id {
			t.Fatalf("run in %s has wrong scope or status: %+v", projectId, stored)
		}
		if !strings.Contains(strings.Join(runner.environment, "\n"), secretData+"@git.example.test") {
			t.Fatalf("run in %s did not receive shared Git credential", projectId)
		}
		runIds[projectId] = created.Run.Id
	}
	if _, err := runs.PipelineRun(ctx, projectA, runIds[projectB]); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("Project A loaded Project B run: %v", err)
	}
	if _, err := runs.PipelineRun(ctx, projectB, runIds[projectA]); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("Project B loaded Project A run: %v", err)
	}
}
