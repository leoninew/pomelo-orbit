import type {
	PaginatedResp,
	PipelineSnapshot,
	PipelineSnapshotListItem,
	PipelineTemplate,
	PipelineTemplateCreateReq,
	PipelineTemplateUpdateReq,
} from '@/types/api';
import request from '@/utils/request';

export const pipelineTemplateApi = {
	list(params?: { page?: number; per_page?: number }): Promise<PaginatedResp<PipelineTemplate>> {
		return request.get('/api/v1/ci/templates', { params });
	},

	get(id: string): Promise<PipelineTemplate> {
		return request.get(`/api/v1/ci/templates/${id}`);
	},

	create(data: PipelineTemplateCreateReq): Promise<PipelineTemplate> {
		return request.post('/api/v1/ci/templates', data);
	},

	update(id: string, data: PipelineTemplateUpdateReq): Promise<PipelineTemplate> {
		return request.put(`/api/v1/ci/templates/${id}`, data);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/v1/ci/templates/${id}`);
	},

	listSnapshots(templateId: string): Promise<PipelineSnapshotListItem[]> {
		return request.get(`/api/v1/ci/templates/${templateId}/snapshots`);
	},

	getSnapshot(snapshotId: string): Promise<PipelineSnapshot> {
		return request.get(`/api/v1/ci/snapshots/${snapshotId}`);
	},
};
