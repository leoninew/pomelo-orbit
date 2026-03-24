/**
 * 通用类型定义
 */

// 导出 API 类型
export * from './api';

// 保留原有的通用类型
export interface ApiResponse<T> {
	data: T
	message?: string
}
