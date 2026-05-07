// 用户相关类型
export interface User {
  id: number
  username: string
  created_at?: string
}

// API 响应类型
export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

// 分页类型
export interface Pagination {
  page: number
  page_size: number
  total: number
}

export interface PaginatedResponse<T> {
  items: T[]
  pagination: Pagination
}
