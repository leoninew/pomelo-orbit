import type {
	PaginatedResp,
	PipelineRun,
	PipelineRunTriggerReq,
	Repository,
	RepositoryCreateReq,
	RepositoryUpdateReq,
} from '@/types/api';
import request from '@/utils/request';

export const repositoryApi = {
	list(params?: { page?: number; per_page?: number }): Promise<PaginatedResp<Repository>> {
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

	delete(id: string): Promise<void> {
		return request.delete(`/api/ci/repository/${id}`);
	},

	trigger(id: string, data?: PipelineRunTriggerReq): Promise<PipelineRun> {
		return request.post(`/api/ci/repository/${id}/trigger`, data || {});
	},
};
