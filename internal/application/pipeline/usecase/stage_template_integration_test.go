package pipelinesvc

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	pipelinedto "github.com/leoninew/pomelo-orbit/internal/application/pipeline/dto"
	"github.com/leoninew/pomelo-orbit/internal/config"
	databasepkg "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	applicationrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/application"
	pipelinerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/pipeline"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
	vcsrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/repository"
	testseed "github.com/leoninew/pomelo-orbit/internal/testutil/seed"
)

const (
	pipelineTemplateUpdateUserID    = "01KKX2YNPF6VJ9N7QYCWG61KVK"
	pipelineTemplateUpdateProjectID = "01KRRKK0K3T519ZQZES3M4QA9Z"
)

func TestApplyPipelineStageTemplateUpdateWritesLatestVersionToOwningPipeline(t *testing.T) {
	service, store, database := newPipelineTemplateUpdateService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	templateVersion := 2
	template := model.PipelineStage{
		Id:          "template-stage-build",
		ProjectId:   pipelineTemplateUpdateProjectID,
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
		ProjectId:            stringPointer(pipelineTemplateUpdateProjectID),
		Kind:                 model.PipelineKindTemplate,
		Name:                 "Build template",
		Description:          "",
		VariableDeclarations: "[]",
		Version:              1,
	}
	if err := store.CreatePipeline(ctx, templatePipeline); err != nil {
		t.Fatalf("create template pipeline: %v", err)
	}
	if err := store.UpdateTemplatePipelineWithReferences(ctx, templatePipeline, []model.PipelineStageReference{{
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

	sourcePipelineID, sourceTemplateName, repositoryID, repositoryName := "source-template", "Source template", "repository-1", "Repository"
	sourcePipelineVersion, sourceStageVersion, sortOrder := 1, 1, 0
	applicationPipeline := model.Pipeline{
		Id:                    "application-pipeline-update",
		ProjectId:             stringPointer(pipelineTemplateUpdateProjectID),
		Kind:                  model.PipelineKindApplication,
		SourcePipelineId:      &sourcePipelineID,
		SourceTemplateName:    &sourceTemplateName,
		SourceTemplateVersion: &sourcePipelineVersion,
		RepositoryId:          &repositoryID,
		RepositoryName:        &repositoryName,
		Name:                  "Application build",
		Description:           "",
		VariableDeclarations:  "[]",
		Version:               1,
	}
	if err := store.CreateApplicationPipelineWithStages(ctx, applicationPipeline, []model.PipelineStage{{
		Id:                             "application-pipeline-stage-build",
		ProjectId:                      pipelineTemplateUpdateProjectID,
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
		SortOrder:                      &sortOrder,
	}}); err != nil {
		t.Fatalf("create application pipeline: %v", err)
	}

	cases := []struct {
		name       string
		pipelineID string
		stageID    string
		stored     func() (int, error)
	}{
		{
			name:       "template pipeline reference",
			pipelineID: templatePipeline.Id,
			stageID:    "template-pipeline-stage-build",
			stored: func() (int, error) {
				references, err := store.TemplatePipelineStageReferences(ctx, templatePipeline.Id)
				if err != nil || len(references) != 1 {
					return 0, err
				}
				return references[0].SourceTemplateStageVersion, nil
			},
		},
		{
			name:       "application pipeline stage",
			pipelineID: applicationPipeline.Id,
			stageID:    "application-pipeline-stage-build",
			stored: func() (int, error) {
				stages, err := store.ApplicationPipelineStages(ctx, applicationPipeline.Id)
				if err != nil || len(stages) != 1 || stages[0].SourceTemplateStageVersion == nil {
					return 0, err
				}
				return *stages[0].SourceTemplateStageVersion, nil
			},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			detail, err := service.ApplyPipelineStageTemplateUpdate(ctx, pipelineTemplateUpdateUserID, testCase.pipelineID, testCase.stageID, pipelinedto.PipelineStageTemplateApplyUpdateInput{
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
	testseed.ApplySQLiteSystemSeed(t, database)
	pipelineStore := pipelinerepo.NewRepository(database)
	service := New(
		projectrepo.NewRepository(database),
		pipelineStore,
		applicationrepo.NewRepository(database),
		vcsrepo.NewRepository(database),
		nil,
	)
	return service, pipelineStore, database
}
