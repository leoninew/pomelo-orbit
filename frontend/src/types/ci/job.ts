// Job
export interface Job {
	id: string
	pipeline_run_id: string
	name: string
	status: 'waiting' | 'running' | 'success' | 'failed' | 'faulted' | 'skipped' | 'canceled'
	started_at?: string
	finished_at?: string
	error_message?: string
}

// JobLog
export interface JobLog {
	id: string
	job_id: string
	content: string
	created_at: string
}

// Artifact
export interface Artifact {
	id: string
	pipeline_run_id: string
	job_name: string
	type: 'docker_image' | 'file'
	name: string
	path?: string
	created_at: string
}

export const jobStatusColors: Record<string, string> = {
	waiting: 'default',
	running: 'processing',
	success: 'success',
	failed: 'error',
	faulted: 'error',
	skipped: 'warning',
	canceled: 'warning',
};
