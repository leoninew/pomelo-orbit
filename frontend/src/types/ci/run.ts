// PipelineRun
export interface PipelineRun {
	id: string
	project_id: string
	trigger: string
	trigger_ref: string
	status: 'waiting' | 'running' | 'success' | 'failed' | 'canceled'
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

export const pipelineRunStatusColors: Record<string, string> = {
	waiting: 'default',
	running: 'processing',
	success: 'success',
	failed: 'error',
	canceled: 'warning',
};
