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
	if (!ms) return '-';
	if (ms < 1000) return `${ms}ms`;
	if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
	return `${(ms / 60000).toFixed(1)}min`;
}

const RUN_BADGE_MAP: Record<string, string> = {
	success: 'badge-outline badge-success',
	failed: 'badge-outline badge-error',
	running: 'badge-outline badge-info',
	waiting: 'badge-outline badge-warning',
	canceled: 'badge-ghost',
};

export function runBadgeClass(status: string): string {
	return RUN_BADGE_MAP[status] ?? 'badge-ghost';
}

const STAGE_RUN_BADGE_MAP: Record<string, string> = {
	success: 'badge-success',
	failed: 'badge-error',
	faulted: 'badge-error',
	running: 'badge-info',
	waiting: 'badge-warning',
	pending: 'badge-warning',
	skipped: 'badge-ghost',
	canceled: 'badge-ghost',
	mixed: 'badge-ghost',
};

export function stageRunBadgeClass(status: string): string {
	return STAGE_RUN_BADGE_MAP[status] ?? 'badge-ghost';
}

/** @deprecated use stageRunBadgeClass */
export const stageBadgeClass = stageRunBadgeClass;
