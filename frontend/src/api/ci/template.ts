import type {
	PaginatedResp,
	PipelineSnapshot,
	PipelineTemplate,
	PipelineTemplateCreateReq,
	PipelineTemplateUpdateReq,
} from '@/types/api';
import request from '@/utils/request';

export const pipelineTemplateApi = {
	list(params?: { page?: number; per_page?: number }): Promise<PaginatedResp<PipelineTemplate>> {
		return request.get('/api/ci/templates', { params });
	},

	get(id: string): Promise<PipelineTemplate> {
		return request.get(`/api/ci/templates/${id}`);
	},

	create(data: PipelineTemplateCreateReq): Promise<PipelineTemplate> {
		return request.post('/api/ci/templates', data);
	},

	update(id: string, data: PipelineTemplateUpdateReq): Promise<PipelineTemplate> {
		return request.put(`/api/ci/templates/${id}`, data);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/ci/templates/${id}`);
	},

	getSnapshot(snapshotId: string): Promise<PipelineSnapshot> {
		return request.get(`/api/ci/snapshots/${snapshotId}`);
	},
};
