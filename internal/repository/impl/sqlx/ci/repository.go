package ci

import (
	"context"
	"fmt"
	"strings"
	"time"

	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"

	"github.com/jmoiron/sqlx"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

var _ repository.CIStore = Repository{}
var _ repository.PipelineExecutionStore = Repository{}

type Repository struct {
	db     *sqlx.DB
	driver string
}

func NewRepository(db *sqlx.DB, driver string) Repository {
	return Repository{db: db, driver: driver}
}

func (r Repository) Project(ctx context.Context, id string) (model.Project, error) {
	var project model.Project
	err := r.db.GetContext(ctx, &project, `SELECT id, name, code, is_active, created_at, updated_at FROM project WHERE id = ?`, id)
	if err != nil {
		return model.Project{}, fmt.Errorf("load project %s: %w", id, err)
	}
	return project, nil
}

func (r Repository) IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error) {
	var count int
	if err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM project_member WHERE project_id = ? AND user_id = ?`, projectId, userId); err != nil {
		return false, fmt.Errorf("check project member %s/%s: %w", projectId, userId, err)
	}
	return count > 0, nil
}

func (r Repository) ListRepositories(ctx context.Context, projectId *string, page int, perPage int, search string) (repository.Page[model.Repository], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	where, args := projectSearchWhere(projectId, search, []string{"name", "code", "repository_url"})
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM repository`+where, args...); err != nil {
		return repository.Page[model.Repository]{}, fmt.Errorf("count repositories: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []model.Repository
	err := r.db.SelectContext(ctx, &items, `SELECT id, project_id, name, code, repository_url, git_credential_id,
		variable_overrides, default_branch, created_at, updated_at FROM repository`+where+` ORDER BY created_at DESC, id LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return repository.Page[model.Repository]{}, fmt.Errorf("list repositories: %w", err)
	}
	return repository.Page[model.Repository]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (r Repository) Repository(ctx context.Context, id string) (model.Repository, error) {
	var repo model.Repository
	err := r.db.GetContext(ctx, &repo, `SELECT id, project_id, name, code, repository_url, git_credential_id,
		variable_overrides, default_branch, created_at, updated_at FROM repository WHERE id = ?`, id)
	if err != nil {
		return model.Repository{}, fmt.Errorf("load repository %s: %w", id, err)
	}
	return repo, nil
}

func (r Repository) RepositoryByCode(ctx context.Context, projectId *string, code string) (model.Repository, error) {
	var repo model.Repository
	where := " WHERE code = ?"
	args := []any{code}
	if projectId != nil {
		where += " AND project_id = ?"
		args = append(args, *projectId)
	}
	err := r.db.GetContext(ctx, &repo, `SELECT id, project_id, name, code, repository_url, git_credential_id,
		variable_overrides, default_branch, created_at, updated_at FROM repository`+where, args...)
	if err != nil {
		return model.Repository{}, fmt.Errorf("load repository by code %s: %w", code, err)
	}
	return repo, nil
}

func (r Repository) CreateRepository(ctx context.Context, repo model.Repository) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO repository (id, project_id, name, code, repository_url, git_credential_id, variable_overrides, default_branch, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)), repo.Id, repo.ProjectId, repo.Name, repo.Code, repo.RepositoryURL, repo.GitCredentialId, repo.VariableOverrides, repo.DefaultBranch)
	if err != nil {
		return fmt.Errorf("create repository %s: %w", repo.Code, err)
	}
	return nil
}

func (r Repository) UpdateRepository(ctx context.Context, repo model.Repository) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE repository SET name = ?, repository_url = ?, git_credential_id = ?, variable_overrides = ?, default_branch = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)),
		repo.Name, repo.RepositoryURL, repo.GitCredentialId, repo.VariableOverrides, repo.DefaultBranch, repo.Id)
	if err != nil {
		return fmt.Errorf("update repository %s: %w", repo.Id, err)
	}
	return nil
}

func (r Repository) DeleteRepository(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM repository WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete repository %s: %w", id, err)
	}
	return nil
}

func (r Repository) RepositoryHasRunningPipelines(ctx context.Context, repositoryId string) (bool, error) {
	var count int
	if err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM pipeline_run WHERE repository_id = ? AND status IN (?, ?)`, repositoryId, status.WorkStatusWaitingToRun, status.WorkStatusRunning); err != nil {
		return false, fmt.Errorf("count running repository pipelines %s: %w", repositoryId, err)
	}
	return count > 0, nil
}

func (r Repository) ListRepositoryWebhooks(ctx context.Context, repositoryId string) ([]model.RepositoryWebhook, error) {
	var items []model.RepositoryWebhook
	err := r.db.SelectContext(ctx, &items, `SELECT id, repository_id, name, template_id, branch_filter, encrypted_secret, enabled, created_at, updated_at FROM repository_webhook WHERE repository_id = ? ORDER BY created_at DESC, id`, repositoryId)
	if err != nil {
		return nil, fmt.Errorf("list repository webhooks %s: %w", repositoryId, err)
	}
	return items, nil
}

func (r Repository) RepositoryWebhook(ctx context.Context, id string) (model.RepositoryWebhook, error) {
	var webhook model.RepositoryWebhook
	err := r.db.GetContext(ctx, &webhook, `SELECT id, repository_id, name, template_id, branch_filter, encrypted_secret, enabled, created_at, updated_at FROM repository_webhook WHERE id = ?`, id)
	if err != nil {
		return model.RepositoryWebhook{}, fmt.Errorf("load repository webhook %s: %w", id, err)
	}
	return webhook, nil
}

func (r Repository) CreateRepositoryWebhook(ctx context.Context, webhook model.RepositoryWebhook) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO repository_webhook (id, repository_id, name, template_id, branch_filter, encrypted_secret, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)), webhook.Id, webhook.RepositoryId, webhook.Name, webhook.TemplateId, webhook.BranchFilter, webhook.EncryptedSecret, webhook.Enabled)
	if err != nil {
		return fmt.Errorf("create repository webhook %s: %w", webhook.Name, err)
	}
	return nil
}

func (r Repository) UpdateRepositoryWebhook(ctx context.Context, webhook model.RepositoryWebhook) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE repository_webhook SET name = ?, template_id = ?, branch_filter = ?, encrypted_secret = ?, enabled = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)), webhook.Name, webhook.TemplateId, webhook.BranchFilter, webhook.EncryptedSecret, webhook.Enabled, webhook.Id)
	if err != nil {
		return fmt.Errorf("update repository webhook %s: %w", webhook.Id, err)
	}
	return nil
}

func (r Repository) DeleteRepositoryWebhook(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM repository_webhook WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete repository webhook %s: %w", id, err)
	}
	return nil
}

func (r Repository) ListCredentials(ctx context.Context, projectId string, page int, perPage int, search string) (repository.Page[model.Credential], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	where, args := credentialWhere(projectId, search)
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM credential`+where, args...); err != nil {
		return repository.Page[model.Credential]{}, fmt.Errorf("count credentials: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []model.Credential
	err := r.db.SelectContext(ctx, &items, `SELECT id, project_id, name, type, encrypted_data, created_at FROM credential`+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return repository.Page[model.Credential]{}, fmt.Errorf("list credentials: %w", err)
	}
	return repository.Page[model.Credential]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (r Repository) Credential(ctx context.Context, id string) (model.Credential, error) {
	var credential model.Credential
	err := r.db.GetContext(ctx, &credential, `SELECT id, project_id, name, type, encrypted_data, created_at FROM credential WHERE id = ?`, id)
	if err != nil {
		return model.Credential{}, fmt.Errorf("load credential %s: %w", id, err)
	}
	return credential, nil
}

func (r Repository) CredentialByName(ctx context.Context, projectId string, name string) (model.Credential, error) {
	var credential model.Credential
	err := r.db.GetContext(ctx, &credential, `SELECT id, project_id, name, type, encrypted_data, created_at FROM credential WHERE project_id = ? AND name = ?`, strings.TrimSpace(projectId), strings.TrimSpace(name))
	if err != nil {
		return model.Credential{}, fmt.Errorf("load credential by name %s: %w", name, err)
	}
	return credential, nil
}

func (r Repository) CredentialExists(ctx context.Context, id string) (bool, error) {
	var count int
	if err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM credential WHERE id = ?`, id); err != nil {
		return false, fmt.Errorf("check credential %s: %w", id, err)
	}
	return count > 0, nil
}

func (r Repository) CredentialName(ctx context.Context, id string) (*string, error) {
	if id == "" {
		return nil, nil
	}
	var name string
	if err := r.db.GetContext(ctx, &name, `SELECT name FROM credential WHERE id = ?`, id); err != nil {
		return nil, fmt.Errorf("load credential name %s: %w", id, err)
	}
	return &name, nil
}

func (r Repository) CreateCredential(ctx context.Context, credential model.Credential) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO credential (id, project_id, name, type, encrypted_data, created_at) VALUES (?, ?, ?, ?, ?, %s)`, db.NowExpr(r.driver)), credential.Id, credential.ProjectId, credential.Name, credential.Type, credential.EncryptedData)
	if err != nil {
		return fmt.Errorf("create credential %s: %w", credential.Name, err)
	}
	return nil
}

func (r Repository) UpdateCredential(ctx context.Context, credential model.Credential) error {
	_, err := r.db.ExecContext(ctx, `UPDATE credential SET name = ?, encrypted_data = ? WHERE id = ?`, credential.Name, credential.EncryptedData, credential.Id)
	if err != nil {
		return fmt.Errorf("update credential %s: %w", credential.Id, err)
	}
	return nil
}

func (r Repository) DeleteCredential(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM credential WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete credential %s: %w", id, err)
	}
	return nil
}

func (r Repository) CredentialReferencedByRepositories(ctx context.Context, projectId string, credentialId string) (bool, error) {
	var count int
	if err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM repository WHERE project_id = ? AND git_credential_id = ?`, strings.TrimSpace(projectId), credentialId); err != nil {
		return false, fmt.Errorf("check credential references %s: %w", credentialId, err)
	}
	return count > 0, nil
}

func (r Repository) ListPipelineTemplates(ctx context.Context, projectId string, page int, perPage int, search string) (repository.Page[model.PipelineTemplate], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	where, args := projectSearchWhere(&projectId, search, []string{"name"})
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM pipeline_template`+where, args...); err != nil {
		return repository.Page[model.PipelineTemplate]{}, fmt.Errorf("count pipeline templates: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []model.PipelineTemplate
	err := r.db.SelectContext(ctx, &items, `SELECT id, project_id, name, description, variable_declarations, version, created_at, updated_at
		FROM pipeline_template`+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return repository.Page[model.PipelineTemplate]{}, fmt.Errorf("list pipeline templates: %w", err)
	}
	return repository.Page[model.PipelineTemplate]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (r Repository) PipelineTemplate(ctx context.Context, id string) (model.PipelineTemplate, error) {
	var template model.PipelineTemplate
	err := r.db.GetContext(ctx, &template, `SELECT id, project_id, name, description, variable_declarations, version, created_at, updated_at FROM pipeline_template WHERE id = ?`, id)
	if err != nil {
		return model.PipelineTemplate{}, fmt.Errorf("load pipeline template %s: %w", id, err)
	}
	return template, nil
}

func (r Repository) PipelineTemplateByName(ctx context.Context, projectId string, name string) (model.PipelineTemplate, error) {
	var template model.PipelineTemplate
	err := r.db.GetContext(ctx, &template, `SELECT id, project_id, name, description, variable_declarations, version, created_at, updated_at FROM pipeline_template WHERE project_id = ? AND name = ?`, projectId, name)
	if err != nil {
		return model.PipelineTemplate{}, fmt.Errorf("load pipeline template by name %s: %w", name, err)
	}
	return template, nil
}

func (r Repository) PipelineTemplateStages(ctx context.Context, templateId string) ([]model.PipelineTemplateStage, error) {
	var stages []model.PipelineTemplateStage
	err := r.db.SelectContext(ctx, &stages, `SELECT id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order FROM pipeline_template_stage WHERE template_id = ? ORDER BY sort_order, id`, templateId)
	if err != nil {
		return nil, fmt.Errorf("load pipeline template stages %s: %w", templateId, err)
	}
	return stages, nil
}

func (r Repository) CreatePipelineTemplate(ctx context.Context, template model.PipelineTemplate) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO pipeline_template (id, project_id, name, description, variable_declarations, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)), template.Id, template.ProjectId, template.Name, template.Description, template.VariableDeclarations, template.Version)
	if err != nil {
		return fmt.Errorf("create pipeline template %s: %w", template.Name, err)
	}
	return nil
}

func (r Repository) UpdatePipelineTemplate(ctx context.Context, template model.PipelineTemplate) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE pipeline_template SET name = ?, description = ?, variable_declarations = ?, version = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)), template.Name, template.Description, template.VariableDeclarations, template.Version, template.Id)
	if err != nil {
		return fmt.Errorf("update pipeline template %s: %w", template.Id, err)
	}
	return nil
}

func (r Repository) UpdatePipelineTemplateWithStages(ctx context.Context, template model.PipelineTemplate, stages []model.PipelineTemplateStage) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update pipeline template %s: %w", template.Id, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE pipeline_template SET name = ?, description = ?, variable_declarations = ?, version = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)), template.Name, template.Description, template.VariableDeclarations, template.Version, template.Id); err != nil {
		return fmt.Errorf("update pipeline template %s: %w", template.Id, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM pipeline_template_stage WHERE template_id = ?`, template.Id); err != nil {
		return fmt.Errorf("delete pipeline template stages %s: %w", template.Id, err)
	}
	for _, stage := range stages {
		if _, err := tx.ExecContext(ctx, `INSERT INTO pipeline_template_stage (id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?)`, stage.Id, stage.TemplateId, stage.StageId, stage.StageName, stage.StageVersion, stage.DependsOn, stage.SortOrder); err != nil {
			return fmt.Errorf("insert pipeline template stage %s: %w", stage.StageId, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update pipeline template %s: %w", template.Id, err)
	}
	committed = true
	return nil
}

func (r Repository) DuplicatePipelineTemplate(ctx context.Context, template model.PipelineTemplate, stages []model.PipelineTemplateStage) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin duplicate pipeline template %s: %w", template.Id, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO pipeline_template (id, project_id, name, description, variable_declarations, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)), template.Id, template.ProjectId, template.Name, template.Description, template.VariableDeclarations, template.Version); err != nil {
		return fmt.Errorf("duplicate pipeline template %s: %w", template.Name, err)
	}
	for _, stage := range stages {
		if _, err := tx.ExecContext(ctx, `INSERT INTO pipeline_template_stage (id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?)`, stage.Id, stage.TemplateId, stage.StageId, stage.StageName, stage.StageVersion, stage.DependsOn, stage.SortOrder); err != nil {
			return fmt.Errorf("duplicate pipeline template stage %s: %w", stage.StageId, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit duplicate pipeline template %s: %w", template.Id, err)
	}
	committed = true
	return nil
}

func (r Repository) PipelineTemplateReferencedByWebhooks(ctx context.Context, templateId string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM repository_webhook WHERE template_id = ?`, templateId)
	if err != nil {
		return false, fmt.Errorf("count pipeline template webhook references %s: %w", templateId, err)
	}
	return count > 0, nil
}

func (r Repository) DeletePipelineTemplate(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM pipeline_template WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete pipeline template %s: %w", id, err)
	}
	return nil
}

func (r Repository) ListBuildStages(ctx context.Context, projectId string, page int, perPage int, search string) (repository.Page[model.BuildStage], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	where, args := projectSearchWhere(&projectId, search, []string{"name"})
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM build_stage`+where, args...); err != nil {
		return repository.Page[model.BuildStage]{}, fmt.Errorf("count build stages: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []model.BuildStage
	err := r.db.SelectContext(ctx, &items, `SELECT id, project_id, name, image, script, artifacts, description, version, created_at, updated_at
		FROM build_stage`+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return repository.Page[model.BuildStage]{}, fmt.Errorf("list build stages: %w", err)
	}
	return repository.Page[model.BuildStage]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (r Repository) BuildStage(ctx context.Context, id string) (model.BuildStage, error) {
	var stage model.BuildStage
	err := r.db.GetContext(ctx, &stage, `SELECT id, project_id, name, image, script, artifacts, description, version, created_at, updated_at FROM build_stage WHERE id = ?`, id)
	if err != nil {
		return model.BuildStage{}, fmt.Errorf("load build stage %s: %w", id, err)
	}
	return stage, nil
}

func (r Repository) BuildStageByName(ctx context.Context, projectId string, name string) (model.BuildStage, error) {
	var stage model.BuildStage
	err := r.db.GetContext(ctx, &stage, `SELECT id, project_id, name, image, script, artifacts, description, version, created_at, updated_at FROM build_stage WHERE project_id = ? AND name = ?`, projectId, name)
	if err != nil {
		return model.BuildStage{}, fmt.Errorf("load build stage by name %s: %w", name, err)
	}
	return stage, nil
}

func (r Repository) BuildStagesByIds(ctx context.Context, projectId string, ids []string) ([]model.BuildStage, error) {
	if len(ids) == 0 {
		return []model.BuildStage{}, nil
	}
	query, args, err := sqlx.In(`SELECT id, project_id, name, image, script, artifacts, description, version, created_at, updated_at FROM build_stage WHERE project_id = ? AND id IN (?)`, projectId, ids)
	if err != nil {
		return nil, fmt.Errorf("build stage ids query: %w", err)
	}
	query = r.db.Rebind(query)
	var stages []model.BuildStage
	if err := r.db.SelectContext(ctx, &stages, query, args...); err != nil {
		return nil, fmt.Errorf("load build stages by ids: %w", err)
	}
	return stages, nil
}

func (r Repository) CreateBuildStage(ctx context.Context, stage model.BuildStage) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO build_stage (id, project_id, name, image, script, artifacts, description, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)), stage.Id, stage.ProjectId, stage.Name, stage.Image, stage.Script, stage.Artifacts, stage.Description, stage.Version)
	if err != nil {
		return fmt.Errorf("create build stage %s: %w", stage.Name, err)
	}
	return nil
}

func (r Repository) UpdateBuildStage(ctx context.Context, stage model.BuildStage) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE build_stage SET name = ?, image = ?, script = ?, artifacts = ?, description = ?, version = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)), stage.Name, stage.Image, stage.Script, stage.Artifacts, stage.Description, stage.Version, stage.Id)
	if err != nil {
		return fmt.Errorf("update build stage %s: %w", stage.Id, err)
	}
	return nil
}

func (r Repository) DeleteBuildStage(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM build_stage WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete build stage %s: %w", id, err)
	}
	return nil
}

func (r Repository) BuildStageReferencedByTemplates(ctx context.Context, projectId string, stageId string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM pipeline_template_stage JOIN pipeline_template ON pipeline_template.id = pipeline_template_stage.template_id WHERE pipeline_template.project_id = ? AND pipeline_template_stage.stage_id = ?`, projectId, stageId)
	if err != nil {
		return false, fmt.Errorf("count build stage template references %s: %w", stageId, err)
	}
	return count > 0, nil
}

func (r Repository) ListPipelineRuns(ctx context.Context, projectId string, repositoryId string, templateId string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (repository.Page[model.PipelineRun], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	where, args := pipelineRunWhere(projectId, repositoryId, templateId, dateFrom, dateTo)
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM pipeline_run`+where, args...); err != nil {
		return repository.Page[model.PipelineRun]{}, fmt.Errorf("count pipeline runs: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []model.PipelineRun
	err := r.db.SelectContext(ctx, &items, fmt.Sprintf(`SELECT id, project_id, repository_id, repository_name, snapshot_id, template_id,
		template_name, template_version, %s, trigger_ref, variables_snapshot, status, retry_of,
		started_at, finished_at, error_message, created_at FROM pipeline_run`+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`, db.QuoteIdent(r.driver, "trigger")), args...)
	if err != nil {
		return repository.Page[model.PipelineRun]{}, fmt.Errorf("list pipeline runs: %w", err)
	}
	return repository.Page[model.PipelineRun]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (r Repository) ListPipelineRunsByRepository(ctx context.Context, repositoryId string, page int, perPage int) (repository.Page[model.PipelineRun], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM pipeline_run WHERE repository_id = ?`, repositoryId); err != nil {
		return repository.Page[model.PipelineRun]{}, fmt.Errorf("count repository pipeline runs %s: %w", repositoryId, err)
	}
	var items []model.PipelineRun
	err := r.db.SelectContext(ctx, &items, fmt.Sprintf(`SELECT id, project_id, repository_id, repository_name, snapshot_id, template_id,
		template_name, template_version, %s, trigger_ref, variables_snapshot, status, retry_of,
		started_at, finished_at, error_message, created_at FROM pipeline_run WHERE repository_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, db.QuoteIdent(r.driver, "trigger")), repositoryId, perPage, (page-1)*perPage)
	if err != nil {
		return repository.Page[model.PipelineRun]{}, fmt.Errorf("list repository pipeline runs %s: %w", repositoryId, err)
	}
	return repository.Page[model.PipelineRun]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (r Repository) PipelineRun(ctx context.Context, id string) (model.PipelineRun, error) {
	var run model.PipelineRun
	err := r.db.GetContext(ctx, &run, fmt.Sprintf(`SELECT id, project_id, repository_id, repository_name, snapshot_id, template_id,
		template_name, template_version, %s, trigger_ref, variables_snapshot, status, retry_of,
		started_at, finished_at, error_message, created_at FROM pipeline_run WHERE id = ?`, db.QuoteIdent(r.driver, "trigger")), id)
	if err != nil {
		return model.PipelineRun{}, fmt.Errorf("load pipeline run %s: %w", id, err)
	}
	return run, nil
}

func (r Repository) CreatePipelineRun(ctx context.Context, run model.PipelineRun) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO pipeline_run (id, project_id, repository_id, repository_name, snapshot_id, template_id, template_name, template_version, %s, trigger_ref, variables_snapshot, status, retry_of, started_at, finished_at, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL, NULL, %s)`, db.QuoteIdent(r.driver, "trigger"), db.NowExpr(r.driver)),
		run.Id, run.ProjectId, run.RepositoryId, run.RepositoryName, run.SnapshotId, run.TemplateId, run.TemplateName, run.TemplateVersion, run.Trigger, run.TriggerRef, run.VariablesSnapshot, run.Status, run.RetryOf)
	if err != nil {
		return fmt.Errorf("create pipeline run %s: %w", run.Id, err)
	}
	return nil
}

func (r Repository) CancelPipelineRun(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE pipeline_run SET status = ?, finished_at = %s WHERE id = ?`, db.NowExpr(r.driver)), status.WorkStatusCanceled, id)
	if err != nil {
		return fmt.Errorf("cancel pipeline run %s: %w", id, err)
	}
	return nil
}

func (r Repository) MarkPipelineRunRunning(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE pipeline_run SET status = ?, started_at = %s, error_message = NULL WHERE id = ?`, db.NowExpr(r.driver)), status.WorkStatusRunning, id)
	if err != nil {
		return fmt.Errorf("mark pipeline run running %s: %w", id, err)
	}
	return nil
}

func (r Repository) CompletePipelineRun(ctx context.Context, id string, status string, message string) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE pipeline_run SET status = ?, finished_at = %s, error_message = NULLIF(?, '') WHERE id = ?`, db.NowExpr(r.driver)), status, message, id)
	if err != nil {
		return fmt.Errorf("complete pipeline run %s: %w", id, err)
	}
	return nil
}

func (r Repository) PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error) {
	var snapshot model.PipelineSnapshot
	err := r.db.GetContext(ctx, &snapshot, `SELECT id, project_id, template_id, version, stages_snapshot,
		variables_snapshot, created_at FROM pipeline_snapshot WHERE id = ?`, id)
	if err != nil {
		return model.PipelineSnapshot{}, fmt.Errorf("load pipeline snapshot %s: %w", id, err)
	}
	return snapshot, nil
}

func (r Repository) LatestPipelineSnapshot(ctx context.Context, templateId string) (model.PipelineSnapshot, error) {
	var snapshot model.PipelineSnapshot
	err := r.db.GetContext(ctx, &snapshot, `SELECT id, project_id, template_id, version, stages_snapshot, variables_snapshot, created_at FROM pipeline_snapshot WHERE template_id = ? ORDER BY version DESC LIMIT 1`, templateId)
	if err != nil {
		return model.PipelineSnapshot{}, fmt.Errorf("load latest pipeline snapshot %s: %w", templateId, err)
	}
	return snapshot, nil
}

func (r Repository) CreatePipelineSnapshot(ctx context.Context, snapshot model.PipelineSnapshot) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO pipeline_snapshot (id, project_id, template_id, version, stages_snapshot, variables_snapshot, created_at)
		VALUES (?, ?, ?, ?, ?, ?, %s)`, db.NowExpr(r.driver)), snapshot.Id, snapshot.ProjectId, snapshot.TemplateId, snapshot.Version, snapshot.StagesSnapshot, snapshot.VariablesSnapshot)
	if err != nil {
		return fmt.Errorf("create pipeline snapshot %s: %w", snapshot.Id, err)
	}
	return nil
}

func (r Repository) ListStageRuns(ctx context.Context, runId string) ([]model.StageRun, error) {
	var items []model.StageRun
	err := r.db.SelectContext(ctx, &items, `SELECT id, pipeline_run_id, stage_id, stage_name, status, started_at, finished_at, exit_code, error_message FROM stage_run WHERE pipeline_run_id = ? ORDER BY rowid`, runId)
	if err != nil {
		return nil, fmt.Errorf("list stage runs %s: %w", runId, err)
	}
	return items, nil
}

func (r Repository) StageRun(ctx context.Context, id string) (model.StageRun, error) {
	var item model.StageRun
	err := r.db.GetContext(ctx, &item, `SELECT id, pipeline_run_id, stage_id, stage_name, status, started_at, finished_at, exit_code, error_message FROM stage_run WHERE id = ?`, id)
	if err != nil {
		return model.StageRun{}, fmt.Errorf("load stage run %s: %w", id, err)
	}
	return item, nil
}

func (r Repository) InsertStageRun(ctx context.Context, stage model.StageRun) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO stage_run (id, pipeline_run_id, stage_id, stage_name, status, started_at, finished_at, exit_code, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, stage.Id, stage.PipelineRunId, stage.StageId, stage.StageName, stage.Status, stage.StartedAt, stage.FinishedAt, stage.ExitCode, stage.ErrorMessage)
	if err != nil {
		return fmt.Errorf("insert stage run %s: %w", stage.Id, err)
	}
	return nil
}

func (r Repository) UpdateStageRun(ctx context.Context, stage model.StageRun) error {
	_, err := r.db.ExecContext(ctx, `UPDATE stage_run SET status = ?, started_at = ?, finished_at = ?, exit_code = ?, error_message = ? WHERE id = ?`, stage.Status, stage.StartedAt, stage.FinishedAt, stage.ExitCode, stage.ErrorMessage, stage.Id)
	if err != nil {
		return fmt.Errorf("update stage run %s: %w", stage.Id, err)
	}
	return nil
}

func (r Repository) ListArtifacts(ctx context.Context, projectId string, repositoryId string, templateId string, page int, perPage int, search string) (repository.Page[model.Artifact], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	where, args := artifactWhere(projectId, repositoryId, templateId, search)
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM artifact`+where, args...); err != nil {
		return repository.Page[model.Artifact]{}, fmt.Errorf("count artifacts: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []model.Artifact
	err := r.db.SelectContext(ctx, &items, `SELECT id, project_id, pipeline_run_id, repository_id, repository_name, template_id, template_name, stage_name, type, name, path, created_at
		FROM artifact`+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return repository.Page[model.Artifact]{}, fmt.Errorf("list artifacts: %w", err)
	}
	return repository.Page[model.Artifact]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (r Repository) ListArtifactsByRun(ctx context.Context, projectId *string, runId string) ([]model.Artifact, error) {
	clauses := []string{"pipeline_run_id = ?"}
	args := []any{runId}
	if projectId != nil && strings.TrimSpace(*projectId) != "" {
		clauses = append(clauses, "project_id = ?")
		args = append(args, strings.TrimSpace(*projectId))
	}
	var items []model.Artifact
	err := r.db.SelectContext(ctx, &items, `SELECT id, project_id, pipeline_run_id, repository_id, repository_name, template_id, template_name, stage_name, type, name, path, created_at
		FROM artifact WHERE `+strings.Join(clauses, " AND ")+` ORDER BY created_at, id`, args...)
	if err != nil {
		return nil, fmt.Errorf("list run artifacts %s: %w", runId, err)
	}
	return items, nil
}

func (r Repository) InsertArtifact(ctx context.Context, projectId *string, run model.PipelineRun, stageName string, artifact model.ArtifactConfig, path string) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO artifact (id, project_id, pipeline_run_id, repository_id, repository_name, template_id, template_name, stage_name, type, name, path, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, %s)`, db.NowExpr(r.driver)), idutil.NewId(), projectId, run.Id, run.RepositoryId, run.RepositoryName, run.TemplateId, run.TemplateName, stageName, artifact.Type, artifact.Name, path)
	if err != nil {
		return fmt.Errorf("insert artifact %s: %w", artifact.Name, err)
	}
	return nil
}

func projectSearchWhere(projectId *string, search string, searchColumns []string) (string, []any) {
	clauses := []string{}
	args := []any{}
	if projectId != nil && strings.TrimSpace(*projectId) != "" {
		clauses = append(clauses, "project_id = ?")
		args = append(args, strings.TrimSpace(*projectId))
	}
	search = strings.TrimSpace(search)
	if search != "" && len(searchColumns) > 0 {
		parts := make([]string, 0, len(searchColumns))
		like := "%" + search + "%"
		for _, column := range searchColumns {
			parts = append(parts, column+" LIKE ?")
			args = append(args, like)
		}
		clauses = append(clauses, "("+strings.Join(parts, " OR ")+")")
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func credentialWhere(projectId string, search string) (string, []any) {
	clauses := []string{"project_id = ?"}
	args := []any{strings.TrimSpace(projectId)}
	search = strings.TrimSpace(search)
	if search != "" {
		like := "%" + search + "%"
		clauses = append(clauses, "(name LIKE ? OR type LIKE ?)")
		args = append(args, like, like)
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func pipelineRunWhere(projectId string, repositoryId string, templateId string, dateFrom *time.Time, dateTo *time.Time) (string, []any) {
	clauses := []string{"project_id = ?"}
	args := []any{strings.TrimSpace(projectId)}
	if strings.TrimSpace(repositoryId) != "" {
		clauses = append(clauses, "repository_id = ?")
		args = append(args, strings.TrimSpace(repositoryId))
	}
	if strings.TrimSpace(templateId) != "" {
		clauses = append(clauses, "template_id = ?")
		args = append(args, strings.TrimSpace(templateId))
	}
	if dateFrom != nil {
		clauses = append(clauses, "created_at >= ?")
		args = append(args, *dateFrom)
	}
	if dateTo != nil {
		clauses = append(clauses, "created_at < ?")
		args = append(args, *dateTo)
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func artifactWhere(projectId string, repositoryId string, templateId string, search string) (string, []any) {
	clauses := []string{"project_id = ?"}
	args := []any{strings.TrimSpace(projectId)}
	if strings.TrimSpace(repositoryId) != "" {
		clauses = append(clauses, "repository_id = ?")
		args = append(args, strings.TrimSpace(repositoryId))
	}
	if strings.TrimSpace(templateId) != "" {
		clauses = append(clauses, "template_id = ?")
		args = append(args, strings.TrimSpace(templateId))
	}
	if strings.TrimSpace(search) != "" {
		like := "%" + strings.TrimSpace(search) + "%"
		clauses = append(clauses, "(name LIKE ? OR path LIKE ?)")
		args = append(args, like, like)
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
