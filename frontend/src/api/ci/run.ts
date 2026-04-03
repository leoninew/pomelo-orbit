import type { Artifact, Job, PaginatedResp, PipelineRun } from '@/types/api';
import request from '@/utils/request';

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

	listJobs(runId: string): Promise<Job[]> {
		return request.get(`/api/ci/runs/${runId}/jobs`);
	},

	listArtifacts(runId: string): Promise<Artifact[]> {
		return request.get(`/api/ci/runs/${runId}/artifacts`);
	},
};
