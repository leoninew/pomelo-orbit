import type {
	PipelineStage,
	PipelineStageCreateReq,
	PipelineStageUpdateReq,
} from '@/types/ci/template';
import request from '@/utils/request';

export const pipelineStageApi = {
	list(): Promise<PipelineStage[]> {
		return request.get('/api/ci/pipeline-stage');
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
