import type {
  PipelineRunArtifactListResp,
  PipelineRunCancelReq,
  PipelineRunPaginatedResp,
  PipelineRunResp,
  PipelineRunRetryReq,
  PipelineStageLogResp,
  PipelineRunTriggerReq,
  PipelineRunVariablePreviewReq,
  PipelineRunVariablePreviewResp,
} from '@/gen/proto/orbit/v1/pipeline_run/pipeline_run';
import request from '@/utils/request';

// PipelineRun API
export const pipelineRunApi = {
  list(params?: {
    page?: number;
    per_page?: number;
    repository_id?: string;
    pipeline_id?: string;
    date_from?: string;
    date_to?: string;
    project_id?: string;
  }): Promise<PipelineRunPaginatedResp> {
    return request.get('/api/pipeline-run', { params });
  },

  get(id: string): Promise<PipelineRunResp> {
    return request.get(`/api/pipeline-run/${id}`);
  },

  trigger(id: string, data: PipelineRunTriggerReq): Promise<PipelineRunResp> {
    return request.post(`/api/pipeline/${id}/trigger`, data);
  },

  previewVariables(
    id: string,
    data: PipelineRunVariablePreviewReq
  ): Promise<PipelineRunVariablePreviewResp> {
    return request.post(`/api/pipeline/${id}/variable-preview`, data);
  },

  retry(id: string, data: PipelineRunRetryReq): Promise<PipelineRunResp> {
    return request.post(`/api/pipeline-run/${id}/retry`, data);
  },

  cancel(id: string, data: PipelineRunCancelReq): Promise<PipelineRunResp> {
    return request.post(`/api/pipeline-run/${id}/cancel`, data);
  },

  listArtifacts(runId: string): Promise<PipelineRunArtifactListResp> {
    return request.get(`/api/pipeline-run/${runId}/artifact`);
  },

  getStageLog(runId: string, stageRunId: string, offset: number): Promise<PipelineStageLogResp> {
    return request.get(`/api/pipeline-run/${runId}/stage/${stageRunId}/log`, {
      params: { offset },
    });
  },
};
