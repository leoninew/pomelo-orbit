const APP_STATUS_COLOR: Record<string, string> = {
	deployed: 'success',
	deploy_failed: 'error',
	deploying: 'processing',
}

const APP_STATUS_LABEL: Record<string, string> = {
	deployed: '运行中',
	deploy_failed: '部署失败',
	deploying: '部署中',
}

export const appStatusColor = (status: string): string => APP_STATUS_COLOR[status] ?? 'default'
export const appStatusLabel = (status: string): string => APP_STATUS_LABEL[status] ?? '未部署'

export function formatDuration(ms?: number | null): string {
	if (!ms) return '-'
	if (ms < 1000) return `${ms}ms`
	if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
	return `${(ms / 60000).toFixed(1)}min`
}
