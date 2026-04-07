import type { TaskStatus } from '../common';

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
}

export interface PipelineRunTriggerReq {
	trigger_ref?: string
	variables?: Record<string, string>
}
