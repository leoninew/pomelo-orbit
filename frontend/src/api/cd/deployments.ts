import type { Deployment, DeploymentDetail, PaginatedResp } from '@/types/api';
import request from '@/utils/request';

// 部署记录相关 API
export const deploymentApi = {
	// 获取部署记录列表
	list(params?: {
		page?: number
		per_page?: number
		application_id?: string
		status?: string
		search?: string
		date_from?: string
		date_to?: string
	}): Promise<PaginatedResp<Deployment>> {
		return request.get('/api/cd/deployments', { params });
	},

	// 获取部署记录详情
	get(id: string): Promise<DeploymentDetail> {
		return request.get(`/api/cd/deployments/${id}`);
	},

	// 取消部署
	cancel(id: string): Promise<void> {
		return request.post(`/api/cd/deployments/${id}/cancel`);
	},

	// 获取部署日志（增量读取）
	getLogs(
		id: string,
		offset: number = 0
	): Promise<{
		logs: string
		offset: number
		is_complete: boolean
		status: string
	}> {
		return request.get(`/api/cd/deployments/${id}/logs`, {
			params: { offset },
		});
	},
};
