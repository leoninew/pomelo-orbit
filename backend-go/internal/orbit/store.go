package orbit

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"backend/internal/db"
)

type Store struct {
	db     *sqlx.DB
	driver string
}

func NewStore(db *sqlx.DB, driver string) Store {
	return Store{db: db, driver: driver}
}

func (s Store) PipelineRun(ctx context.Context, id string) (PipelineRun, error) {
	var run PipelineRun
	err := s.db.GetContext(ctx, &run, fmt.Sprintf(`SELECT id, project_id, repository_id, repository_name, snapshot_id, template_id,
		template_name, template_version, %s, trigger_ref, variables_snapshot, status, retry_of,
		started_at, finished_at, error_message, created_at FROM pipeline_run WHERE id = ?`, db.QuoteIdent(s.driver, "trigger")), id)
	if err != nil {
		return PipelineRun{}, fmt.Errorf("load pipeline run %s: %w", id, err)
	}
	return run, nil
}

func (s Store) Repository(ctx context.Context, id string) (Repository, error) {
	var repo Repository
	err := s.db.GetContext(ctx, &repo, `SELECT id, project_id, name, code, repository_url, git_credential_id,
		variable_overrides, default_branch, created_at, updated_at FROM repository WHERE id = ?`, id)
	if err != nil {
		return Repository{}, fmt.Errorf("load repository %s: %w", id, err)
	}
	return repo, nil
}

func (s Store) PipelineSnapshot(ctx context.Context, id string) (PipelineSnapshot, error) {
	var snapshot PipelineSnapshot
	err := s.db.GetContext(ctx, &snapshot, `SELECT id, project_id, template_id, version, stages_snapshot,
		variables_snapshot, created_at FROM pipeline_snapshot WHERE id = ?`, id)
	if err != nil {
		return PipelineSnapshot{}, fmt.Errorf("load pipeline snapshot %s: %w", id, err)
	}
	return snapshot, nil
}

func (s Store) Credential(ctx context.Context, id string) (Credential, error) {
	var credential Credential
	err := s.db.GetContext(ctx, &credential, `SELECT id, project_id, name, type, encrypted_data, created_at FROM credential WHERE id = ?`, id)
	if err != nil {
		return Credential{}, fmt.Errorf("load credential %s: %w", id, err)
	}
	return credential, nil
}

func (s Store) MarkPipelineRunRunning(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE pipeline_run SET status = ?, started_at = %s, error_message = NULL WHERE id = ?`, db.NowExpr(s.driver)), WorkStatusRunning, id)
	if err != nil {
		return fmt.Errorf("mark pipeline run running %s: %w", id, err)
	}
	return nil
}

func (s Store) CompletePipelineRun(ctx context.Context, id string, status string, message string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE pipeline_run SET status = ?, finished_at = %s, error_message = NULLIF(?, '') WHERE id = ?`, db.NowExpr(s.driver)), status, message, id)
	if err != nil {
		return fmt.Errorf("complete pipeline run %s: %w", id, err)
	}
	return nil
}

func (s Store) InsertStageRun(ctx context.Context, stage StageRun) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO stage_run (id, pipeline_run_id, stage_id, stage_name, status, started_at, finished_at, exit_code, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, stage.Id, stage.PipelineRunId, stage.StageId, stage.StageName, stage.Status, stage.StartedAt, stage.FinishedAt, stage.ExitCode, stage.ErrorMessage)
	if err != nil {
		return fmt.Errorf("insert stage run %s: %w", stage.Id, err)
	}
	return nil
}

func (s Store) UpdateStageRun(ctx context.Context, stage StageRun) error {
	_, err := s.db.ExecContext(ctx, `UPDATE stage_run SET status = ?, started_at = ?, finished_at = ?, exit_code = ?, error_message = ? WHERE id = ?`, stage.Status, stage.StartedAt, stage.FinishedAt, stage.ExitCode, stage.ErrorMessage, stage.Id)
	if err != nil {
		return fmt.Errorf("update stage run %s: %w", stage.Id, err)
	}
	return nil
}

func (s Store) InsertArtifact(ctx context.Context, projectId *string, run PipelineRun, stageName string, artifact ArtifactConfig, path string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO artifact (id, project_id, pipeline_run_id, repository_id, repository_name, template_id, template_name, stage_name, type, name, path, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, %s)`, db.NowExpr(s.driver)), NewId(), projectId, run.Id, run.RepositoryId, run.RepositoryName, run.TemplateId, run.TemplateName, stageName, artifact.Type, artifact.Name, path)
	if err != nil {
		return fmt.Errorf("insert artifact %s: %w", artifact.Name, err)
	}
	return nil
}

func (s Store) Application(ctx context.Context, id string) (Application, error) {
	var app Application
	err := s.db.GetContext(ctx, &app, `SELECT id, project_id, name, code, image_pull_policy, status, route_managed, created_at, updated_at FROM application WHERE id = ?`, id)
	if err != nil {
		return Application{}, fmt.Errorf("load application %s: %w", id, err)
	}
	return app, nil
}

func (s Store) Deployment(ctx context.Context, id string) (Deployment, error) {
	var deployment Deployment
	err := s.db.GetContext(ctx, &deployment, `SELECT id, project_id, application_id, application_name, operation_type, trigger_type, status,
		started_at, finished_at, duration_ms, log_text, error_message, is_rollback, rollback_from_deployment_id FROM deployment WHERE id = ?`, id)
	if err != nil {
		return Deployment{}, fmt.Errorf("load deployment %s: %w", id, err)
	}
	return deployment, nil
}

func (s Store) ConfigFiles(ctx context.Context, applicationId string) ([]ApplicationConfigFile, error) {
	var files []ApplicationConfigFile
	err := s.db.SelectContext(ctx, &files, `SELECT id, application_id, path, content, created_at, updated_at FROM application_config_file WHERE application_id = ? ORDER BY path`, applicationId)
	if err != nil {
		return nil, fmt.Errorf("load config files for application %s: %w", applicationId, err)
	}
	return files, nil
}

func (s Store) ServiceConfigs(ctx context.Context, applicationId string) ([]ApplicationServiceConfig, error) {
	var configs []ApplicationServiceConfig
	err := s.db.SelectContext(ctx, &configs, `SELECT id, application_id, service_name, image, environment, volumes, created_at, updated_at FROM application_service WHERE application_id = ?`, applicationId)
	if err != nil {
		return nil, fmt.Errorf("load service configs for application %s: %w", applicationId, err)
	}
	return configs, nil
}

func (s Store) Routes(ctx context.Context, applicationId string) ([]ApplicationRoute, error) {
	var routes []ApplicationRoute
	err := s.db.SelectContext(ctx, &routes, `SELECT id, application_id, service_name, domain, port, created_at, updated_at FROM application_route WHERE application_id = ?`, applicationId)
	if err != nil {
		return nil, fmt.Errorf("load routes for application %s: %w", applicationId, err)
	}
	return routes, nil
}

func (s Store) MarkApplicationStatus(ctx context.Context, id string, status string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE application SET status = ?, updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)), status, id)
	if err != nil {
		return fmt.Errorf("mark application %s status: %w", id, err)
	}
	return nil
}

func (s Store) MarkDeploymentRunning(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE deployment SET status = ?, started_at = %s, error_message = NULL WHERE id = ?`, db.NowExpr(s.driver)), WorkStatusRunning, id)
	if err != nil {
		return fmt.Errorf("mark deployment running %s: %w", id, err)
	}
	return nil
}

func (s Store) CompleteDeployment(ctx context.Context, id string, status string, message string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE deployment SET status = ?, finished_at = %s, duration_ms = %s, error_message = NULLIF(?, '') WHERE id = ?`, db.NowExpr(s.driver), db.DurationMillisExpr(s.driver, "started_at")), status, message, id)
	if err != nil {
		return fmt.Errorf("complete deployment %s: %w", id, err)
	}
	return nil
}
