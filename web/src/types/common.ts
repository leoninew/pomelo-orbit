// 通用列表响应
export interface ListResp<T> {
  items: T[];
}

// 通用分页响应
export interface PaginatedResp<T> {
  items: T[];
  total: number;
  page: number;
  per_page: number;
  pages: number;
}

// 通用错误响应
export interface ErrorResp<T> {
  detail: T;
}

// 异步任务状态（CI PipelineRun、CD Deployment、StageRun 统一使用）
export type TaskStatus =
  | 'waiting_to_run'
  | 'running'
  | 'ran_to_completion'
  | 'faulted'
  | 'canceled';
