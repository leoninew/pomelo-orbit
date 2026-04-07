// StageRun — Stage 执行记录
export interface StageRun {
	id: string
	pipeline_run_id: string
	name: string
	status: 'waiting' | 'running' | 'success' | 'failed' | 'faulted' | 'skipped' | 'canceled'
	started_at?: string
	finished_at?: string
	exit_code?: number
	error_message?: string
}

// StageLog — Stage 执行日志
export interface StageLog {
	id: string
	stage_run_id: string
	content: string
	created_at: string
}

// Artifact
export interface Artifact {
	id: string
	pipeline_run_id: string
	stage_name: string
	type: 'docker_image' | 'file'
	name: string
	path?: string
	created_at: string
}
