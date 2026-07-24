package pipelinerunsvc

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	pipelinerundto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/executionlog"
	queuedispatch "gitee.com/leoninew/PomeloOrbit-go/internal/queue/dispatch"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
	credentialrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/credential"
	pipelinerepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/pipeline"
	pipelinerunrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/pipeline_run"
	projectrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/project"
	vcsrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/repository"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/task"
)

const (
	ciTestUserId       = "01KKX2YNPF6VJ9N7QYCWG61KVK"
	ciTestProjectId    = "01KRRKK0K3T519ZQZES3M4QA9Z"
	ciTestRepositoryId = "01KNNRBH52BQJYT9487B2H8N62"
	ciTestTemplateId   = "01KNVEJPWVK757139NMNNNCEFE"
	ciTestSecretKey    = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
)

func TestRepositoryTriggerCreatesAndReusesCurrentTemplateSnapshot(t *testing.T) {
	service, database := newPipelineRunIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	first, err := service.TriggerRepository(ctx, ciTestUserId, pipelinerundto.PipelineRunTriggerInput{RepositoryId: ciTestRepositoryId, TemplateId: ciTestTemplateId, TriggerRef: "main", Variables: map[string]string{"working_dir": "."}})
	if err != nil {
		t.Fatal(err)
	}
	if first.Run.Id == "" || first.Run.RepositoryId != ciTestRepositoryId || first.Run.Status != status.WorkStatusWaitingToRun || first.Run.TemplateVersion != 7 {
		t.Fatalf("unexpected triggered run: %+v", first.Run)
	}
	if first.Run.SnapshotId == "" {
		t.Fatal("expected triggered run to reference a snapshot")
	}
	var snapshotCount int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM pipeline_snapshot WHERE template_id = ? AND version = 7`, ciTestTemplateId).Scan(&snapshotCount); err != nil {
		t.Fatal(err)
	}
	if snapshotCount != 1 {
		t.Fatalf("expected one current template snapshot, got %d", snapshotCount)
	}
	var taskPayload string
	if err := database.QueryRowContext(ctx, `SELECT payload_json FROM background_task ORDER BY created_at DESC LIMIT 1`).Scan(&taskPayload); err != nil {
		t.Fatal(err)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(taskPayload), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["pipeline_run_id"] != first.Run.Id {
		t.Fatalf("unexpected task payload: %+v", payload)
	}

	second, err := service.TriggerRepository(ctx, ciTestUserId, pipelinerundto.PipelineRunTriggerInput{RepositoryId: ciTestRepositoryId, TemplateId: ciTestTemplateId, TriggerRef: "main", Variables: map[string]string{"working_dir": "."}})
	if err != nil {
		t.Fatal(err)
	}
	if second.Run.SnapshotId != first.Run.SnapshotId {
		t.Fatalf("expected existing current snapshot to be reused, got %q want %q", second.Run.SnapshotId, first.Run.SnapshotId)
	}
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM pipeline_snapshot WHERE template_id = ? AND version = 7`, ciTestTemplateId).Scan(&snapshotCount); err != nil {
		t.Fatal(err)
	}
	if snapshotCount != 1 {
		t.Fatalf("expected existing current snapshot only, got %d", snapshotCount)
	}
}

func TestPipelineRunServiceReadsLogsAndFiltersArtifacts(t *testing.T) {
	service, database := newPipelineRunIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	insertCIPipelineRunTestData(t, database, "run-log-test", status.WorkStatusRunning)
	insertCIPipelineStageRunTestData(t, database, "stage-log-test", "run-log-test", status.WorkStatusRunning)
	logPath := service.workspace.StageLogPath("run-log-test", "stage-log-test")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	logResp, err := service.PipelineStageLog(ctx, ciTestUserId, "run-log-test", "stage-log-test", 6)
	if err != nil {
		t.Fatal(err)
	}
	if logResp.Logs != "world\n" || logResp.Offset != 12 || logResp.IsComplete {
		t.Fatalf("unexpected stage log response: %+v", logResp)
	}

	if _, err := database.ExecContext(ctx, `INSERT INTO artifact (id, project_id, pipeline_run_id, repository_id, repository_name, template_id, template_name, stage_name, type, name, path) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, "artifact-log-test", ciTestProjectId, "run-log-test", ciTestRepositoryId, "golang/example", ciTestTemplateId, "Go 构建流水线", "test", "file", "test.log", "test.log"); err != nil {
		t.Fatal(err)
	}
	artifacts, err := service.ListPipelineRunArtifacts(ctx, ciTestUserId, "run-log-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 || artifacts[0].Id != "artifact-log-test" || artifacts[0].RepositoryId != ciTestRepositoryId {
		t.Fatalf("unexpected run artifacts: %+v", artifacts)
	}

	filtered, err := service.ListArtifacts(ctx, ciTestUserId, pipelinerundto.ArtifactListInput{ProjectId: ciTestProjectId, RepositoryId: ciTestRepositoryId, TemplateId: ciTestTemplateId, Search: "test.log", Page: 1, PerPage: 20})
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Total != 1 || len(filtered.Items) != 1 || filtered.Items[0].Id != "artifact-log-test" {
		t.Fatalf("unexpected filtered artifacts: %+v", filtered)
	}
	if _, err := service.ListArtifacts(ctx, ciTestUserId, pipelinerundto.ArtifactListInput{ProjectId: ciTestProjectId, RepositoryId: "missing-repository"}); err == nil || apperror.StatusCode(err) != http.StatusNotFound {
		t.Fatalf("expected missing repository filter to return 404, got %v", err)
	}
	if _, err := service.ListArtifacts(ctx, ciTestUserId, pipelinerundto.ArtifactListInput{ProjectId: ciTestProjectId, TemplateId: "missing-template"}); err == nil || apperror.StatusCode(err) != http.StatusNotFound {
		t.Fatalf("expected missing template filter to return 404, got %v", err)
	}
}

func newPipelineRunIntegrationService(t *testing.T) (Service, *sql.DB) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	tasks := tasksvc.New(taskrepo.NewRepository(database), 3)
	service := New(
		projectrepo.NewRepository(database),
		credentialrepo.NewRepository(database),
		vcsrepo.NewRepository(database),
		pipelinerepo.NewRepository(database),
		pipelinerunrepo.NewRepository(database),
		queuedispatch.NewPipelineRunDispatcher(tasks),
		newTestWorkspace(t),
		ciTestSecretKey,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		fakeContainerRunner{},
		executionlog.Store{},
	)
	return service, database
}

func insertCIPipelineRunTestData(t *testing.T, database *sql.DB, id string, runStatus string) {
	t.Helper()
	if _, err := database.Exec(`INSERT INTO pipeline_run (id, project_id, repository_id, repository_name, snapshot_id, template_id, template_name, template_version, trigger, trigger_ref, variables_snapshot, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, ciTestProjectId, ciTestRepositoryId, "golang/example", "snapshot-"+id, ciTestTemplateId, "Go 构建流水线", 7, "manual", "main", "{}", runStatus); err != nil {
		t.Fatal(err)
	}
}

func insertCIPipelineStageRunTestData(t *testing.T, database *sql.DB, id string, runId string, stageStatus string) {
	t.Helper()
	if _, err := database.Exec(`INSERT INTO pipeline_stage_run (id, pipeline_run_id, stage_id, stage_name, status) VALUES (?, ?, ?, ?, ?)`, id, runId, "stage-1", "test", stageStatus); err != nil {
		t.Fatal(err)
	}
}
