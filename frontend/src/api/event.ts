import type { PaginatedResp, WebhookEvent, WebhookEventDetail } from '@/types/api';
import request from '@/utils/request';

// 回调事件相关 API
export const eventApi = {
	// 获取事件列表
	list(params?: {
		page?: number
		per_page?: number
		search?: string
		source?: string
		status?: string
		date_from?: string
		date_to?: string
	}): Promise<PaginatedResp<WebhookEvent>> {
		return request.get('/api/webhook-event', { params });
	},

	// 获取事件详情
	get(id: string): Promise<WebhookEventDetail> {
		return request.get(`/api/webhook-event/${id}`);
	},
};
