export type BadgeTone = 'default' | 'primary' | 'success' | 'error' | 'warning' | 'info';

const APP_STATUS_TONE: Record<string, BadgeTone> = {
	undeployed: 'default',
	deploying: 'info',
	deployed: 'success',
	deploy_failed: 'error',
};

const APP_STATUS_LABEL: Record<string, string> = {
	undeployed: '未部署',
	deploying: '部署中',
	deployed: '运行中',
	deploy_failed: '部署失败',
};

export function appStatusTone(status: string): BadgeTone {
	return requireStatus(APP_STATUS_TONE, status, 'application');
}

export function appStatusLabel(status: string): string {
	return requireStatus(APP_STATUS_LABEL, status, 'application');
}

const STATUS_LABEL: Record<string, string> = {
	waiting_to_run: '待运行',
	running: '运行中',
	ran_to_completion: '成功',
	faulted: '异常',
	canceled: '已取消',
};

export function statusLabel(status: string): string {
	return requireStatus(STATUS_LABEL, status, 'task');
}

const STATUS_TONE: Record<string, BadgeTone> = {
	waiting_to_run: 'warning',
	running: 'info',
	ran_to_completion: 'success',
	faulted: 'error',
	canceled: 'warning',
};

export function statusTone(status: string): BadgeTone {
	return requireStatus(STATUS_TONE, status, 'task');
}

function requireStatus<T>(map: Record<string, T>, status: string, scope: string): T {
	const value = map[status];
	if (value === undefined) {
		throw new Error(`Unknown ${scope} status: ${status}`);
	}
	return value;
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
	return requireStatus(STATUS_COLOR, status, 'task');
}

export function isTerminalStatus(status: string): boolean {
	return status === 'ran_to_completion' || status === 'faulted' || status === 'canceled';
}
