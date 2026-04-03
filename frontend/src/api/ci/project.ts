import type {
	PaginatedResp,
	PipelineRun,
	PipelineRunTriggerReq,
	Project,
	ProjectCreateReq,
	ProjectUpdateReq,
} from '@/types/api';
import request from '@/utils/request';

export const projectApi = {
	list(params?: { page?: number; per_page?: number }): Promise<PaginatedResp<Project>> {
		return request.get('/api/v1/ci/projects', { params });
	},

	get(id: string): Promise<Project> {
		return request.get(`/api/v1/ci/projects/${id}`);
	},

	create(data: ProjectCreateReq): Promise<Project> {
		return request.post('/api/v1/ci/projects', data);
	},

	update(id: string, data: ProjectUpdateReq): Promise<Project> {
		return request.put(`/api/v1/ci/projects/${id}`, data);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/v1/ci/projects/${id}`);
	},

	trigger(id: string, data?: PipelineRunTriggerReq): Promise<PipelineRun> {
		return request.post(`/api/v1/ci/projects/${id}/trigger`, data || {});
	},
};
