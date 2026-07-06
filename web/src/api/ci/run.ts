import type {
  PipelineRunArtifactListResp,
  PipelineRunPaginatedResp,
  PipelineRunResp,
  PipelineStageLogResp,
} from '@/gen/proto/orbit/api/v1/pipeline_run';
import request from '@/utils/request';

// PipelineRun API
export const pipelineRunApi = {
  list(params?: {
    page?: number;
    per_page?: number;
    repository_id?: string;
    template_id?: string;
    date_from?: string;
    date_to?: string;
    project_id?: string;
  }): Promise<PipelineRunPaginatedResp> {
    return request.get('/api/ci/run', { params });
  },

  get(id: string): Promise<PipelineRunResp> {
    return request.get(`/api/ci/run/${id}`);
  },

  retry(id: string): Promise<PipelineRunResp> {
    return request.post(`/api/ci/run/${id}/retry`, {});
  },

  cancel(id: string): Promise<PipelineRunResp> {
    return request.post(`/api/ci/run/${id}/cancel`, {});
  },

  listArtifacts(runId: string): Promise<PipelineRunArtifactListResp> {
    return request.get(`/api/ci/run/${runId}/artifacts`);
  },

  getStageLog(runId: string, stageRunId: string, offset: number): Promise<PipelineStageLogResp> {
    return request.get(`/api/ci/run/${runId}/stages/${stageRunId}/log`, {
      params: { offset },
    });
  },
};
