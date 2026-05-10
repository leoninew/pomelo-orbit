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
	waiting_to_run:
		'border border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300',
	running:
		'border border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-500/30 dark:bg-blue-500/10 dark:text-blue-300',
	ran_to_completion:
		'border border-green-200 bg-green-50 text-green-700 dark:border-green-500/30 dark:bg-green-500/10 dark:text-green-300',
	faulted:
		'border border-red-200 bg-red-50 text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300',
	canceled: 'border border-border bg-muted text-muted-foreground',
};

export function statusBadgeClass(status: string): string {
	return STATUS_BADGE_MAP[status] ?? 'border border-border bg-muted text-muted-foreground';
}

// 异步任务状态颜色（用于 DAG 节点文字、连接线等需要 hex 颜色的场景）
export const STATUS_COLOR: Record<string, string> = {
	waiting_to_run: '#d97706', // amber-600
	running: '#2563eb', // blue-600
	ran_to_completion: '#16a34a', // green-600
	faulted: '#dc2626', // red-600
	canceled: '#9ca3af', // gray-400
};

const STATUS_COLOR_DEFAULT = '#94a3b8'; // slate-400，无状态时

export function statusColor(status: string | undefined): string {
	if (!status) {
		return STATUS_COLOR_DEFAULT;
	}
	return STATUS_COLOR[status] ?? STATUS_COLOR_DEFAULT;
}

export function isTerminalStatus(status: string): boolean {
	return status === 'ran_to_completion' || status === 'faulted' || status === 'canceled';
}
