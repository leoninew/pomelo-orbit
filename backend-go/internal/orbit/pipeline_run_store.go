package orbit

import (
	"context"
	"fmt"

	"backend/internal/db"
)

func (s Store) ListStageRuns(ctx context.Context, runId string) ([]StageRun, error) {
	var items []StageRun
	err := s.db.SelectContext(ctx, &items, `SELECT id, pipeline_run_id, stage_id, stage_name, status, started_at, finished_at, exit_code, error_message FROM stage_run WHERE pipeline_run_id = ? ORDER BY rowid`, runId)
	if err != nil {
		return nil, fmt.Errorf("list stage runs %s: %w", runId, err)
	}
	return items, nil
}

func (s Store) StageRun(ctx context.Context, id string) (StageRun, error) {
	var item StageRun
	err := s.db.GetContext(ctx, &item, `SELECT id, pipeline_run_id, stage_id, stage_name, status, started_at, finished_at, exit_code, error_message FROM stage_run WHERE id = ?`, id)
	if err != nil {
		return StageRun{}, fmt.Errorf("load stage run %s: %w", id, err)
	}
	return item, nil
}

func (s Store) CancelPipelineRun(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE pipeline_run SET status = ?, finished_at = %s WHERE id = ?`, db.NowExpr(s.driver)), WorkStatusCanceled, id)
	if err != nil {
		return fmt.Errorf("cancel pipeline run %s: %w", id, err)
	}
	return nil
}
