/**
 * 状态样式映射
 *
 * 只负责状态值到视觉样式的映射，不包含文本翻译
 */

export type BadgeTone = 'default' | 'primary' | 'success' | 'error' | 'warning' | 'info';

/**
 * 应用状态 -> Badge 色调
 *
 * 后端枚举: ApplicationStatus
 * - undeployed: 未部署
 * - deploying: 部署中
 * - deployed: 已部署
 * - deploy_failed: 部署失败
 */
export function appStatusTone(status: string): BadgeTone {
  const tones: Record<string, BadgeTone> = {
    undeployed: 'default',
    deploying: 'info',
    deployed: 'success',
    deploy_failed: 'error',
  };
  return tones[status] ?? 'default';
}

/**
 * 任务状态 -> Badge 色调
 *
 * 后端枚举: TaskStatus
 * - waiting_to_run: 待运行
 * - running: 运行中
 * - ran_to_completion: 成功
 * - faulted: 异常
 * - canceled: 已取消
 */
export function statusTone(status: string): BadgeTone {
  const tones: Record<string, BadgeTone> = {
    waiting_to_run: 'warning',
    running: 'info',
    ran_to_completion: 'success',
    faulted: 'error',
    canceled: 'warning',
  };
  return tones[status] ?? 'default';
}

/**
 * 任务状态 -> Hex 颜色（用于 DAG 图节点和连接线）
 */
export function statusColor(status: string | undefined): string {
  if (!status) {
    return '#94a3b8'; // slate-400，无状态时
  }

  const colors: Record<string, string> = {
    waiting_to_run: '#d97706', // amber-600
    running: '#2563eb', // blue-600
    ran_to_completion: '#16a34a', // green-600
    faulted: '#dc2626', // red-600
    canceled: '#9ca3af', // gray-400
  };
  return colors[status] ?? '#94a3b8';
}

/**
 * 判断任务状态是否为终态
 */
export function isTerminalStatus(status: string): boolean {
  return status === 'ran_to_completion' || status === 'faulted' || status === 'canceled';
}
