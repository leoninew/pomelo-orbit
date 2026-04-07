import type {
	PipelineStage,
	PipelineStageCreateReq,
	PipelineStageUpdateReq,
} from '@/types/ci/template';
import request from '@/utils/request';

export const pipelineStageApi = {
	list(): Promise<PipelineStage[]> {
		return request.get('/api/ci/pipeline-stages');
	},

	get(id: string): Promise<PipelineStage> {
		return request.get(`/api/ci/pipeline-stages/${id}`);
	},

	create(data: PipelineStageCreateReq): Promise<PipelineStage> {
		return request.post('/api/ci/pipeline-stages', data);
	},

	update(id: string, data: PipelineStageUpdateReq): Promise<PipelineStage> {
		return request.put(`/api/ci/pipeline-stages/${id}`, data);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/ci/pipeline-stages/${id}`);
	},
};
