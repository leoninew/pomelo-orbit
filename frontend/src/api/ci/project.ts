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
		return request.get('/api/ci/projects', { params });
	},

	get(id: string): Promise<Project> {
		return request.get(`/api/ci/projects/${id}`);
	},

	create(data: ProjectCreateReq): Promise<Project> {
		return request.post('/api/ci/projects', data);
	},

	update(id: string, data: ProjectUpdateReq): Promise<Project> {
		return request.put(`/api/ci/projects/${id}`, data);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/ci/projects/${id}`);
	},

	trigger(id: string, data?: PipelineRunTriggerReq): Promise<PipelineRun> {
		return request.post(`/api/ci/projects/${id}/trigger`, data || {});
	},
};
