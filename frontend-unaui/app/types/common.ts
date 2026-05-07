// 通用分页响应
export interface PaginatedResp<T> {
  items: T[]
  total: number
  page: number
  per_page: number
  pages: number
}

// 异步任务状态
export type TaskStatus = 'waiting_to_run' | 'running' | 'ran_to_completion' | 'faulted' | 'canceled'
