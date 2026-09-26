package pipelinesvc

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	credentialsvc "github.com/leoninew/pomelo-orbit/internal/application/credential/usecase"
	pipelinedto "github.com/leoninew/pomelo-orbit/internal/application/pipeline/dto"
	repositorydto "github.com/leoninew/pomelo-orbit/internal/application/repository/dto"
	repositorysvc "github.com/leoninew/pomelo-orbit/internal/application/repository/usecase"
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

const (
	pipelineTemplateUpdateUserId    = "01KKX2YNPF6VJ9N7QYCWG61KVK"
	pipelineTemplateUpdateProjectId = "01KRRKK0K3T519ZQZES3M4QA9Z"
)

func TestSharedRepositoryAndCredentialInstantiateInAnotherProject(t *testing.T) {
	pipelineService, _, database := newPipelineTemplateUpdateService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	otherProjectId := "shared-resource-project"
	if _, err := database.ExecContext(ctx, `INSERT INTO project (id, name, code) VALUES (?, ?, ?)`, otherProjectId, "Shared resource", "shared-resource"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO project_member (project_id, user_id) VALUES (?, ?)`, otherProjectId, pipelineTemplateUpdateUserId); err != nil {
		t.Fatal(err)
	}

	projectStore := projectrepo.NewRepository(database)
	credentialStore := credentialrepo.NewRepository(database)
	repositoryStore := vcsrepo.NewRepository(database)
	credentialService := credentialsvc.New(projectStore, credentialStore, repositoryStore, "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	repositoryService := repositorysvc.New(projectStore, credentialStore, repositoryStore, nil, nil)
	credential, err := credentialService.CreateCredential(ctx, pipelineTemplateUpdateUserId, credentialdto.CredentialCreateInput{
		ProjectId: pipelineTemplateUpdateProjectId, Name: "Shared Git token", Type: "github_token", Data: "shared-secret",
	})
	if err != nil {
		t.Fatalf("create credential in first project: %v", err)
	}
	repository, err := repositoryService.CreateRepository(ctx, pipelineTemplateUpdateUserId, repositorydto.RepositoryCreateInput{
		ProjectId: pipelineTemplateUpdateProjectId, Name: "Shared source", Code: "shared-source",
		RepositoryType: model.RepositoryTypeRemoteGit, RepositoryUrl: "https://example.invalid/shared.git",
		GitCredentialId: &credential.Id, DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("create repository in first project: %v", err)
	}
	visibleCredential, err := credentialService.CredentialDetailForUser(ctx, pipelineTemplateUpdateUserId, otherProjectId, credential.Id)
	if err != nil || visibleCredential.Data != "shared-secret" {
		t.Fatalf("read shared credential in second project: %+v, %v", visibleCredential, err)
	}
	visibleRepository, err := repositoryService.RepositoryForUser(ctx, pipelineTemplateUpdateUserId, otherProjectId, repository.Repository.Id)
	if err != nil || visibleRepository.GitCredentialName == nil || *visibleRepository.GitCredentialName != credential.Name {
		t.Fatalf("read shared repository in second project: %+v, %v", visibleRepository, err)
	}
	instance, err := pipelineService.InstantiatePipeline(ctx, pipelineTemplateUpdateUserId, otherProjectId, "01M391Y93PEYM4CNTGM5EBRS40", pipelinedto.PipelineInstantiateInput{
		Name: "Other project build", RepositoryId: repository.Repository.Id,
	})
	if err != nil {
		t.Fatalf("instantiate with shared repository: %v", err)
	}
	if instance.Pipeline.ProjectId == nil || *instance.Pipeline.ProjectId != otherProjectId || instance.Pipeline.RepositoryId == nil || *instance.Pipeline.RepositoryId != repository.Repository.Id {
		t.Fatalf("instantiated pipeline has wrong project or repository: %+v", instance.Pipeline)
	}
}

func TestApplyPipelineStageTemplateUpdateWritesLatestVersionToOwningPipeline(t *testing.T) {
	service, store, database := newPipelineTemplateUpdateService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	templateVersion := 2
	template := model.PipelineStage{
		Id:          "template-stage-build",
		ProjectId:   "",
		Kind:        model.PipelineStageKindTemplate,
		Name:        "Build image",
		Image:       "docker:27",
		Script:      "docker build .",
		Artifacts:   stringPointer("[]"),
		Version:     &templateVersion,
		Description: "latest build stage",
	}
	if err := store.CreatePipelineStageTemplate(ctx, template); err != nil {
		t.Fatalf("create template stage: %v", err)
	}

	templatePipeline := model.Pipeline{
		Id:                   "template-pipeline-update",
		ProjectId:            nil,
		Kind:                 model.PipelineKindTemplate,
		Name:                 "Build template",
		Description:          "",
		VariableDeclarations: "[]",
		Version:              1,
	}
	if err := store.CreatePipeline(ctx, templatePipeline); err != nil {
		t.Fatalf("create template pipeline: %v", err)
	}
	if err := store.UpdateTemplatePipelineWithReferences(ctx, pipelineTemplateUpdateProjectId, templatePipeline, []model.PipelineStageReference{{
		Id:                             "template-pipeline-stage-build",
		PipelineId:                     templatePipeline.Id,
		SourceTemplateStageId:          template.Id,
		SourceTemplateStageName:        "Build image",
		SourceTemplateStageVersion:     1,
		SourceTemplateStageDescription: "previous build stage",
		Name:                           "Build",
		Image:                          "docker:26",
		Script:                         "docker build .",
		Description:                    "",
		Artifacts:                      "[]",
		DependsOn:                      "[]",
		SortOrder:                      0,
	}}); err != nil {
		t.Fatalf("create template pipeline reference: %v", err)
	}

	sourcePipelineId, sourceTemplateName, repositoryId, repositoryName := "source-template", "Source template", "repository-1", "Repository"
	sourcePipelineVersion, sourceStageVersion := 1, 1
	if err := vcsrepo.NewRepository(database).CreateRepository(ctx, model.Repository{Id: repositoryId, Name: repositoryName, Code: "repository", RepositoryType: "git", RepositoryUrl: "https://example.invalid/repository.git", DefaultBranch: "main"}); err != nil {
		t.Fatalf("create repository: %v", err)
	}
	applicationPipeline := model.Pipeline{
		Id:                    "application-pipeline-update",
		ProjectId:             stringPointer(pipelineTemplateUpdateProjectId),
		Kind:                  model.PipelineKindApplication,
		SourcePipelineId:      &sourcePipelineId,
		SourceTemplateName:    &sourceTemplateName,
		SourceTemplateVersion: &sourcePipelineVersion,
		RepositoryId:          &repositoryId,
		RepositoryName:        &repositoryName,
		Name:                  "Application build",
		Description:           "",
		VariableDeclarations:  "[]",
		Version:               1,
	}
	if err := store.CreateApplicationPipelineWithStages(ctx, applicationPipeline, []model.PipelineStage{{
		Id:                             "application-pipeline-stage-build",
		ProjectId:                      pipelineTemplateUpdateProjectId,
		Kind:                           model.PipelineStageKindApplication,
		PipelineId:                     &applicationPipeline.Id,
		SourceTemplateStageId:          &template.Id,
		SourceTemplateStageName:        stringPointer("Build image"),
		SourceTemplateStageVersion:     &sourceStageVersion,
		SourceTemplateStageDescription: stringPointer("previous build stage"),
		Name:                           "Build",
		Image:                          "docker:26",
		Script:                         "docker build .",
		Description:                    "",
		Artifacts:                      stringPointer("[]"),
		DependsOn:                      stringPointer("[]"),
		SortOrder:                      0,
	}}); err != nil {
		t.Fatalf("create application pipeline: %v", err)
	}

	cases := []struct {
		name       string
		pipelineId string
		stageId    string
		stored     func() (int, error)
	}{
		{
			name:       "template pipeline reference",
			pipelineId: templatePipeline.Id,
			stageId:    "template-pipeline-stage-build",
			stored: func() (int, error) {
				references, err := store.TemplatePipelineStageReferences(ctx, pipelineTemplateUpdateProjectId, templatePipeline.Id)
				if err != nil || len(references) != 1 {
					return 0, err
				}
				return references[0].SourceTemplateStageVersion, nil
			},
		},
		{
			name:       "application pipeline stage",
			pipelineId: applicationPipeline.Id,
			stageId:    "application-pipeline-stage-build",
			stored: func() (int, error) {
				stages, err := store.ApplicationPipelineStages(ctx, pipelineTemplateUpdateProjectId, applicationPipeline.Id)
				if err != nil || len(stages) != 1 || stages[0].SourceTemplateStageVersion == nil {
					return 0, err
				}
				return *stages[0].SourceTemplateStageVersion, nil
			},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			detail, err := service.ApplyPipelineStageTemplateUpdate(ctx, pipelineTemplateUpdateUserId, pipelineTemplateUpdateProjectId, testCase.pipelineId, testCase.stageId, pipelinedto.PipelineStageTemplateApplyUpdateInput{
				ExpectedSourceTemplateStageVersion: 1,
				TargetTemplateStageVersion:         templateVersion,
			})
			if err != nil {
				t.Fatalf("apply template update: %v", err)
			}
			if len(detail.StageNodes) != 1 || detail.StageNodes[0].LatestTemplateStageVersion != nil {
				t.Fatalf("updated pipeline still reports a pending template update: %#v", detail.StageNodes)
			}
			storedVersion, err := testCase.stored()
			if err != nil {
				t.Fatalf("load stored stage: %v", err)
			}
			if storedVersion != templateVersion {
				t.Fatalf("stored source version = %d, want %d", storedVersion, templateVersion)
			}
		})
	}
}

func TestDeleteApplicationPipelineRequiresFinishedRuns(t *testing.T) {
	service, store, database := newPipelineTemplateUpdateService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	projectId := pipelineTemplateUpdateProjectId
	repositoryId := "delete-run-repository"
	pipelineId := "delete-run-pipeline"
	if err := vcsrepo.NewRepository(database).CreateRepository(ctx, model.Repository{
		Id: repositoryId, Name: "Repository", Code: repositoryId,
		RepositoryType: model.RepositoryTypeRemoteGit, RepositoryUrl: "https://example.invalid/repository.git", DefaultBranch: "main",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreatePipeline(ctx, model.Pipeline{
		Id: pipelineId, ProjectId: &projectId, Kind: model.PipelineKindApplication,
		RepositoryId: &repositoryId, Name: "Delete after run", VariableDeclarations: "[]", Version: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO pipeline_run
		(id, project_id, repository_id, repository_name, snapshot_id, pipeline_id, pipeline_name, pipeline_version, trigger, repository_ref, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"delete-run-1", projectId, repositoryId, "Repository", "snapshot-1", pipelineId, "Delete after run", 1, "manual", "main", "waiting_to_run"); err != nil {
		t.Fatal(err)
	}
	if err := service.DeletePipeline(ctx, pipelineTemplateUpdateUserId, projectId, pipelineId); err == nil || !strings.Contains(err.Error(), "unfinished runs") {
		t.Fatalf("delete with waiting run = %v, want unfinished-run conflict", err)
	}
	if _, err := database.ExecContext(ctx, `UPDATE pipeline_run SET status = 'ran_to_completion' WHERE id = ?`, "delete-run-1"); err != nil {
		t.Fatal(err)
	}
	if err := service.DeletePipeline(ctx, pipelineTemplateUpdateUserId, projectId, pipelineId); err != nil {
		t.Fatalf("delete with completed run: %v", err)
	}
	var runCount int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM pipeline_run WHERE id = ?`, "delete-run-1").Scan(&runCount); err != nil {
		t.Fatal(err)
	}
	if runCount != 1 {
		t.Fatalf("completed run count after pipeline deletion = %d, want 1", runCount)
	}
}

func TestDeletePipelineStageNodeRemovesScopedVariables(t *testing.T) {
	cases := []struct {
		name       string
		pipelineId string
		stageId    string
	}{
		{name: "template", pipelineId: "01M391Y93NTCXQ6J8H34VBMJJR", stageId: "01M38WMQDCY38G5B09PVB697C5"},
		{name: "application", pipelineId: "01M391Y93NTCXQ6J8H38DXEN8Q", stageId: "01M38WMQDCY38G5B09PRXSWZXT"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			service, store, database := newPipelineTemplateUpdateService(t)
			defer func() { _ = database.Close() }()
			ctx := context.Background()

			detail, err := service.DeletePipelineStageNode(ctx, pipelineTemplateUpdateUserId, pipelineTemplateUpdateProjectId, testCase.pipelineId, testCase.stageId)
			if err != nil {
				t.Fatalf("delete build stage: %v", err)
			}
			if len(detail.StageNodes) != 1 || detail.StageNodes[0].Node.Id == testCase.stageId {
				t.Fatalf("remaining stages = %#v", detail.StageNodes)
			}
			if detail.Pipeline.VariableDeclarations != "[]" {
				t.Fatalf("updated pipeline = %#v", detail.Pipeline)
			}
			stored, err := store.Pipeline(ctx, pipelineTemplateUpdateProjectId, testCase.pipelineId)
			if err != nil {
				t.Fatal(err)
			}
			if stored.VariableDeclarations != "[]" {
				t.Fatalf("stored pipeline = %#v", stored)
			}
		})
	}
}

func newPipelineTemplateUpdateService(t *testing.T) (Service, repository.PipelineStore, *sql.DB) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := databasepkg.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		_ = database.Close()
		t.Fatal(err)
	}
	pipelineStore := pipelinerepo.NewRepository(database)
	service := New(
		projectrepo.NewRepository(database),
		pipelineStore,
		pipelinerunrepo.NewRepository(database),
		applicationrepo.NewRepository(database),
		vcsrepo.NewRepository(database),
		nil,
	)
	return service, pipelineStore, database
}
