const APP_STATUS_COLOR: Record<string, string> = {
	deployed: 'success',
	deploy_failed: 'error',
	deploying: 'processing',
};

const APP_STATUS_LABEL: Record<string, string> = {
	deployed: '运行中',
	deploy_failed: '部署失败',
	deploying: '部署中',
};

export const appStatusColor = (status: string): string => APP_STATUS_COLOR[status] ?? 'default';
export const appStatusLabel = (status: string): string => APP_STATUS_LABEL[status] ?? '未部署';

export function formatDuration(ms?: number | null): string {
	if (!ms) {
		return '-';
	}
	if (ms < 1000) {
		return `${ms}ms`;
	}
	if (ms < 60000) {
		return `${(ms / 1000).toFixed(1)}s`;
	}
	return `${(ms / 60000).toFixed(1)}min`;
}

// 异步任务状态标签（适用于 CI PipelineRun、CD Deployment、StageRun 等）
const STATUS_LABEL: Record<string, string> = {
	waiting_to_run: '待运行',
	running: '运行中',
	ran_to_completion: '成功',
	faulted: '异常',
	canceled: '已取消',
};

export function statusLabel(status: string): string {
	return STATUS_LABEL[status] ?? status;
}

// 异步任务状态 badge 样式
const STATUS_BADGE_MAP: Record<string, string> = {
	waiting_to_run: 'badge-outline badge-warning',
	running: 'badge-outline badge-info',
	ran_to_completion: 'badge-outline badge-success',
	faulted: 'badge-outline badge-error',
	canceled: 'badge-ghost',
};

export function statusBadgeClass(status: string): string {
	return STATUS_BADGE_MAP[status] ?? 'badge-ghost';
}

export function isTerminalStatus(status: string): boolean {
	return status === 'ran_to_completion' || status === 'faulted' || status === 'canceled';
}
