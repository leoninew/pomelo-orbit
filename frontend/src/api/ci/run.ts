import type { Artifact, PaginatedResp, PipelineRun } from '@/types/api';
import request from '@/utils/request';

export interface StageLogResp {
	logs: string
	offset: number
	is_complete: boolean
}

// PipelineRun API
export const pipelineRunApi = {
	list(params?: {
		page?: number
		per_page?: number
		project_id?: string
	}): Promise<PaginatedResp<PipelineRun>> {
		return request.get('/api/ci/runs', { params });
	},

	get(id: string): Promise<PipelineRun> {
		return request.get(`/api/ci/runs/${id}`);
	},

	retry(id: string): Promise<PipelineRun> {
		return request.post(`/api/ci/runs/${id}/retry`);
	},

	cancel(id: string): Promise<PipelineRun> {
		return request.post(`/api/ci/runs/${id}/cancel`);
	},

	listArtifacts(runId: string): Promise<Artifact[]> {
		return request.get(`/api/ci/runs/${runId}/artifacts`);
	},

	getStageLog(runId: string, stageRunId: string, offset: number): Promise<StageLogResp> {
		return request.get(`/api/ci/runs/${runId}/stages/${stageRunId}/log`, { params: { offset } });
	},
};
