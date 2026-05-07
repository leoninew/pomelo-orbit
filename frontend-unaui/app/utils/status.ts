import type { TaskStatus } from '~/types/common'

export function statusLabel(status: TaskStatus): string {
  const labels: Record<TaskStatus, string> = {
    waiting_to_run: '等待中',
    running: '运行中',
    ran_to_completion: '成功',
    faulted: '失败',
    canceled: '已取消'
  }
  return labels[status] || status
}

export function statusBadgeVariant(status: TaskStatus): string {
  const variants: Record<TaskStatus, string> = {
    waiting_to_run: 'soft-gray',
    running: 'solid-info',
    ran_to_completion: 'solid-success',
    faulted: 'solid-error',
    canceled: 'soft-gray'
  }
  return variants[status] || 'soft-gray'
}
