import type {
  PipelineRunArtifactListResp,
  PipelineRunCancelReq,
  PipelineRunPaginatedResp,
  PipelineRunResp,
  PipelineRunRetryReq,
  PipelineStageLogResp,
  PipelineRunTriggerReq,
} from '@/gen/proto/orbit/v1/pipeline_run/pipeline_run';
import request, { type AxiosRequestConfig } from '@/utils/request';

// PipelineRun API
export const pipelineRunApi = {
  list(
    projectId: string,
    params?: {
      page?: number;
      per_page?: number;
      repository_id?: string;
      pipeline_id?: string;
      date_from?: string;
      date_to?: string;
    }
  ): Promise<PipelineRunPaginatedResp> {
    return request.get('/api/pipeline-run', { params: { project_id: projectId, ...params } });
  },

  get(projectId: string, id: string): Promise<PipelineRunResp> {
    return request.get(`/api/pipeline-run/${id}`, { params: { project_id: projectId } });
  },

  delete(projectId: string, id: string): Promise<void> {
    return request.delete(`/api/pipeline-run/${id}`, { params: { project_id: projectId } });
  },

  trigger(projectId: string, id: string, data: PipelineRunTriggerReq): Promise<PipelineRunResp> {
    return request.post(`/api/pipeline/${id}/trigger`, data, { params: { project_id: projectId } });
  },

  retry(projectId: string, id: string, data: PipelineRunRetryReq): Promise<PipelineRunResp> {
    return request.post(`/api/pipeline-run/${id}/retry`, data, {
      params: { project_id: projectId },
    });
  },

  cancel(projectId: string, id: string, data: PipelineRunCancelReq): Promise<PipelineRunResp> {
    return request.post(`/api/pipeline-run/${id}/cancel`, data, {
      params: { project_id: projectId },
    });
  },

  listArtifacts(projectId: string, runId: string): Promise<PipelineRunArtifactListResp> {
    return request.get(`/api/pipeline-run/${runId}/artifact`, {
      params: { project_id: projectId },
    });
  },

  getStageLog(
    projectId: string,
    runId: string,
    stageRunId: string,
    offset: number,
    config?: AxiosRequestConfig
  ): Promise<PipelineStageLogResp> {
    return request.get(`/api/pipeline-run/${runId}/stage/${stageRunId}/log`, {
      ...config,
      params: { project_id: projectId, offset },
    });
  },
};
