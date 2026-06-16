package repository

import (
	"context"
	"fmt"

	"backend/internal/db"
)

type RepositoryTrigger struct {
	TemplateId string
	TriggerRef string
	Variables  string
	SnapshotId string
	RunId      string
}

func (s Store) RepositoryByCode(ctx context.Context, projectId *string, code string) (Repository, error) {
	var repo Repository
	where := " WHERE code = ?"
	args := []any{code}
	if projectId != nil {
		where += " AND project_id = ?"
		args = append(args, *projectId)
	}
	err := s.db.GetContext(ctx, &repo, `SELECT id, project_id, name, code, repository_url, git_credential_id,
		variable_overrides, default_branch, created_at, updated_at FROM repository`+where, args...)
	if err != nil {
		return Repository{}, fmt.Errorf("load repository by code %s: %w", code, err)
	}
	return repo, nil
}

func (s Store) CredentialExists(ctx context.Context, id string) (bool, error) {
	var count int
	if err := s.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM credential WHERE id = ?`, id); err != nil {
		return false, fmt.Errorf("check credential %s: %w", id, err)
	}
	return count > 0, nil
}

func (s Store) CredentialName(ctx context.Context, id string) (*string, error) {
	if id == "" {
		return nil, nil
	}
	var name string
	if err := s.db.GetContext(ctx, &name, `SELECT name FROM credential WHERE id = ?`, id); err != nil {
		return nil, fmt.Errorf("load credential name %s: %w", id, err)
	}
	return &name, nil
}

func (s Store) CreateRepository(ctx context.Context, repo Repository) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO repository (id, project_id, name, code, repository_url, git_credential_id, variable_overrides, default_branch, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(s.driver), db.NowExpr(s.driver)), repo.Id, repo.ProjectId, repo.Name, repo.Code, repo.RepositoryURL, repo.GitCredentialId, repo.VariableOverrides, repo.DefaultBranch)
	if err != nil {
		return fmt.Errorf("create repository %s: %w", repo.Code, err)
	}
	return nil
}

func (s Store) UpdateRepository(ctx context.Context, repo Repository) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE repository SET name = ?, repository_url = ?, git_credential_id = ?, variable_overrides = ?, default_branch = ?, updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)),
		repo.Name, repo.RepositoryURL, repo.GitCredentialId, repo.VariableOverrides, repo.DefaultBranch, repo.Id)
	if err != nil {
		return fmt.Errorf("update repository %s: %w", repo.Id, err)
	}
	return nil
}

func (s Store) DeleteRepository(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM repository WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete repository %s: %w", id, err)
	}
	return nil
}

func (s Store) RepositoryHasRunningPipelines(ctx context.Context, repositoryId string) (bool, error) {
	var count int
	if err := s.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM pipeline_run WHERE repository_id = ? AND status IN (?, ?)`, repositoryId, WorkStatusWaitingToRun, WorkStatusRunning); err != nil {
		return false, fmt.Errorf("count running repository pipelines %s: %w", repositoryId, err)
	}
	return count > 0, nil
}

func (s Store) ListRepositoryWebhooks(ctx context.Context, repositoryId string) ([]RepositoryWebhook, error) {
	var items []RepositoryWebhook
	err := s.db.SelectContext(ctx, &items, `SELECT id, repository_id, name, template_id, branch_filter, encrypted_secret, enabled, created_at, updated_at FROM repository_webhook WHERE repository_id = ? ORDER BY created_at DESC, id`, repositoryId)
	if err != nil {
		return nil, fmt.Errorf("list repository webhooks %s: %w", repositoryId, err)
	}
	return items, nil
}

func (s Store) RepositoryWebhook(ctx context.Context, id string) (RepositoryWebhook, error) {
	var webhook RepositoryWebhook
	err := s.db.GetContext(ctx, &webhook, `SELECT id, repository_id, name, template_id, branch_filter, encrypted_secret, enabled, created_at, updated_at FROM repository_webhook WHERE id = ?`, id)
	if err != nil {
		return RepositoryWebhook{}, fmt.Errorf("load repository webhook %s: %w", id, err)
	}
	return webhook, nil
}

func (s Store) CreateRepositoryWebhook(ctx context.Context, webhook RepositoryWebhook) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO repository_webhook (id, repository_id, name, template_id, branch_filter, encrypted_secret, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(s.driver), db.NowExpr(s.driver)), webhook.Id, webhook.RepositoryId, webhook.Name, webhook.TemplateId, webhook.BranchFilter, webhook.EncryptedSecret, webhook.Enabled)
	if err != nil {
		return fmt.Errorf("create repository webhook %s: %w", webhook.Name, err)
	}
	return nil
}

func (s Store) UpdateRepositoryWebhook(ctx context.Context, webhook RepositoryWebhook) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE repository_webhook SET name = ?, template_id = ?, branch_filter = ?, encrypted_secret = ?, enabled = ?, updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)), webhook.Name, webhook.TemplateId, webhook.BranchFilter, webhook.EncryptedSecret, webhook.Enabled, webhook.Id)
	if err != nil {
		return fmt.Errorf("update repository webhook %s: %w", webhook.Id, err)
	}
	return nil
}

func (s Store) DeleteRepositoryWebhook(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM repository_webhook WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete repository webhook %s: %w", id, err)
	}
	return nil
}

func (s Store) LatestPipelineSnapshot(ctx context.Context, templateId string) (PipelineSnapshot, error) {
	var snapshot PipelineSnapshot
	err := s.db.GetContext(ctx, &snapshot, `SELECT id, project_id, template_id, version, stages_snapshot, variables_snapshot, created_at FROM pipeline_snapshot WHERE template_id = ? ORDER BY version DESC LIMIT 1`, templateId)
	if err != nil {
		return PipelineSnapshot{}, fmt.Errorf("load latest pipeline snapshot %s: %w", templateId, err)
	}
	return snapshot, nil
}

func (s Store) PipelineTemplate(ctx context.Context, id string) (PipelineTemplate, error) {
	var template PipelineTemplate
	err := s.db.GetContext(ctx, &template, `SELECT id, project_id, name, description, variable_declarations, version, created_at, updated_at FROM pipeline_template WHERE id = ?`, id)
	if err != nil {
		return PipelineTemplate{}, fmt.Errorf("load pipeline template %s: %w", id, err)
	}
	return template, nil
}

func (s Store) CreatePipelineRun(ctx context.Context, run PipelineRun) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO pipeline_run (id, project_id, repository_id, repository_name, snapshot_id, template_id, template_name, template_version, %s, trigger_ref, variables_snapshot, status, retry_of, started_at, finished_at, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL, NULL, %s)`, db.QuoteIdent(s.driver, "trigger"), db.NowExpr(s.driver)),
		run.Id, run.ProjectId, run.RepositoryId, run.RepositoryName, run.SnapshotId, run.TemplateId, run.TemplateName, run.TemplateVersion, run.Trigger, run.TriggerRef, run.VariablesSnapshot, run.Status, run.RetryOf)
	if err != nil {
		return fmt.Errorf("create pipeline run %s: %w", run.Id, err)
	}
	return nil
}
