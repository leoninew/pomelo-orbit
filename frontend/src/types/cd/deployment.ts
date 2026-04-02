// 部署记录相关
export interface Deployment {
	id: string
	application_id: string
	application_name: string | null
	operation_type: string
	trigger_type: string
	trigger_ref: string | null
	env_file: string | null
	status: string
	started_at: string
	finished_at: string | null
	duration_ms: number | null
	error_message: string | null
}

export interface DeploymentDetail extends Deployment {
	log_text: string | null
}

// 状态颜色映射
export const deploymentStatusColors: Record<string, string> = {
	queued: 'default',
	running: 'processing',
	success: 'success',
	failed: 'error',
};
