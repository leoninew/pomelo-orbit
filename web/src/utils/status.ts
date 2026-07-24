/**
 * 状态样式映射
 *
 * 只负责状态值到视觉样式的映射，不包含文本翻译
 */

export type BadgeTone = 'default' | 'primary' | 'success' | 'error' | 'warning' | 'info';

/**
 * Application kind -> Badge 色调
 *
 * - standard: 中性
 * - gateway: 主色，便于在列表右上角区分
 */
export function applicationKindTone(kind: string | undefined | null): BadgeTone {
  return kind === 'gateway' ? 'primary' : 'default';
}

/**
 * 应用运行态 -> Badge 色调
 *
 * 来自 Service.status（application.service_status）；无 Service 时为空。
 * - '' / undeployed: 从未部署
 * - deploying | running | stopped | faulted
 */
export function appStatusTone(status: string): BadgeTone {
  const tones: Record<string, BadgeTone> = {
    undeployed: 'default',
    deploying: 'info',
    running: 'success',
    stopped: 'warning',
    faulted: 'error',
  };
  return tones[status] ?? 'default';
}

/**
 * Version 生命周期 -> Badge 色调
 *
 * - unpublished: 可编辑
 * - published: 已发布、不可变
 */
export function versionStatusTone(status: string): BadgeTone {
  const tones: Record<string, BadgeTone> = {
    unpublished: 'warning',
    published: 'success',
  };
  return tones[status] ?? 'default';
}

/** 展示用 service_status：空串视为从未部署 */
export function normalizeServiceStatus(status: string | undefined | null): string {
  return status && status.trim() ? status : 'undeployed';
}

/**
 * docker compose 容器 State -> Badge 色调
 *
 * 常见值: running | exited | dead | paused | created | restarting | removing
 */
export function containerStateTone(state: string | undefined | null): BadgeTone {
  const value = (state || '').toLowerCase();
  if (value === 'running' || value === 'up') {
    return 'success';
  }
  if (value === 'exited' || value === 'dead' || value === 'removing') {
    return 'error';
  }
  if (value === 'restarting' || value === 'created') {
    return 'info';
  }
  if (value === 'paused' || value === 'stopped') {
    return 'warning';
  }
  return 'default';
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
