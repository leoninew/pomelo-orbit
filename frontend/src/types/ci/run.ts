import type { TaskStatus } from '../common';
import type { StageRun } from './stage_run';

// PipelineRun
export interface PipelineRun {
	id: string
	project_id: string
	trigger: string
	trigger_ref: string
	status: TaskStatus
	retry_of?: string
	pipeline_snapshot_id: string
	started_at?: string
	finished_at?: string
	created_at: string
	stage_runs: StageRun[]
}

export interface PipelineRunTriggerReq {
	trigger_ref?: string
	variables?: Record<string, string>
}
