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
		return request.get('/api/ci/template', { params });
	},

	get(id: string): Promise<PipelineTemplate> {
		return request.get(`/api/ci/template/${id}`);
	},

	create(data: PipelineTemplateCreateReq): Promise<PipelineTemplate> {
		return request.post('/api/ci/template', data);
	},

	update(id: string, data: PipelineTemplateUpdateReq): Promise<PipelineTemplate> {
		return request.put(`/api/ci/template/${id}`, data);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/ci/template/${id}`);
	},

	getSnapshot(snapshotId: string): Promise<PipelineSnapshot> {
		return request.get(`/api/ci/snapshot/${snapshotId}`);
	},
};
