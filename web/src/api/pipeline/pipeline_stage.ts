import type {
  PipelineStageCreateReq,
  PipelineStagePaginatedResp,
  PipelineStageResp,
  PipelineStageUpdateReq,
} from '@/gen/proto/orbit/v1/pipeline/pipeline_stage';
import request from '@/utils/request';

export const pipelineStageApi = {
  list(
    projectId: string,
    params?: {
      page?: number;
      per_page?: number;
      search?: string;
    }
  ): Promise<PipelineStagePaginatedResp> {
    return request.get('/api/pipeline-stage', { params: { project_id: projectId, ...params } });
  },
  get(projectId: string, id: string): Promise<PipelineStageResp> {
    return request.get(`/api/pipeline-stage/${id}`, { params: { project_id: projectId } });
  },
  create(projectId: string, data: PipelineStageCreateReq): Promise<PipelineStageResp> {
    return request.post('/api/pipeline-stage', data, { params: { project_id: projectId } });
  },
  update(projectId: string, id: string, data: PipelineStageUpdateReq): Promise<PipelineStageResp> {
    return request.put(`/api/pipeline-stage/${id}`, data, { params: { project_id: projectId } });
  },
  delete(projectId: string, id: string): Promise<void> {
    return request.delete(`/api/pipeline-stage/${id}`, { params: { project_id: projectId } });
  },
};
