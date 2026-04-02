// 通用分页响应
export interface PaginatedResp<T> {
	items: T[]
	total: number
	page: number
	per_page: number
	pages: number
}
