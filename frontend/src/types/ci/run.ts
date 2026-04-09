import type { TaskStatus } from '../common';
import type { StageRun } from './stage_run';

// PipelineRun
export interface PipelineRun {
	id: string
	repository_id: string
	repository_name: string
	trigger: string
	trigger_ref: string
	status: TaskStatus
	retry_of?: string
	pipeline_snapshot_id: string
	template_id: string
	template_name: string
	started_at?: string
	finished_at?: string
	created_at: string
	stage_runs: StageRun[]
}

export interface PipelineRunTriggerReq {
	trigger_ref?: string
	variables?: Record<string, string>
}
