import type { TaskStatus } from '../common';
import type { VariableDeclaration } from './template';
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
	snapshot_id: string
	template_id: string
	template_name: string
	template_version: number
	variables_snapshot: VariableDeclaration[]
	started_at?: string
	finished_at?: string
	error_message?: string
	created_at: string
	stage_runs: StageRun[]
}

export interface PipelineRunTriggerReq {
	template_id: string
	trigger_ref?: string
	variables?: Record<string, string>
}
