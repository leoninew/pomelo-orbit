import type {
	PipelineStage,
	PipelineStageCreateReq,
	PipelineStageUpdateReq,
} from '@/types/ci/template';
import type { PaginatedResp } from '@/types/api';
import request from '@/utils/request';

export const pipelineStageApi = {
	list(params?: { page?: number; per_page?: number }): Promise<PaginatedResp<PipelineStage>> {
		return request.get('/api/ci/pipeline-stage', { params });
	},

	get(id: string): Promise<PipelineStage> {
		return request.get(`/api/ci/pipeline-stage/${id}`);
	},

	create(data: PipelineStageCreateReq): Promise<PipelineStage> {
		return request.post('/api/ci/pipeline-stage', data);
	},

	update(id: string, data: PipelineStageUpdateReq): Promise<PipelineStage> {
		return request.put(`/api/ci/pipeline-stage/${id}`, data);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/ci/pipeline-stage/${id}`);
	},

	duplicate(id: string): Promise<PipelineStage> {
		return request.post(`/api/ci/pipeline-stage/${id}/duplicate`);
	},
};
