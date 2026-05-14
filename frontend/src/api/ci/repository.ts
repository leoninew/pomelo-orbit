import type { PipelineRun, PipelineRunTriggerReq } from '@/types/ci/run';
import type { PaginatedResp } from '@/types/common';
import type { Repository, RepositoryCreateReq, RepositoryUpdateReq } from '@/types/ci/repository';
import request from '@/utils/request';

export const repositoryApi = {
	list(params?: {
		page?: number
		per_page?: number
		search?: string
		projectId?: string
	}): Promise<PaginatedResp<Repository>> {
		return request.get('/api/ci/repository', { params });
	},

	get(id: string): Promise<Repository> {
		return request.get(`/api/ci/repository/${id}`);
	},

	create(data: RepositoryCreateReq): Promise<Repository> {
		return request.post('/api/ci/repository', data);
	},

	update(id: string, data: RepositoryUpdateReq): Promise<Repository> {
		return request.put(`/api/ci/repository/${id}`, data);
	},

	delete(id: string, params?: { delete_workspace?: boolean }): Promise<void> {
		return request.delete(`/api/ci/repository/${id}`, { params });
	},

	trigger(id: string, data?: PipelineRunTriggerReq): Promise<PipelineRun> {
		return request.post(`/api/ci/repository/${id}/trigger`, data || {});
	},
};
