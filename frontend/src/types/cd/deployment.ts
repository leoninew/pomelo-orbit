import type { TaskStatus } from '../common';

// 部署记录相关
export interface Deployment {
	id: string
	application_id: string
	application_name: string | null
	operation_type: string
	trigger_type: string
	env_file: string | null
	status: TaskStatus
	started_at: string
	finished_at: string | null
	duration_ms: number | null
	error_message: string | null
}

export interface DeploymentDetail extends Deployment {
	log_text: string | null
}
