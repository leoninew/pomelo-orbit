import type {
  PipelineStageCreateReq,
  PipelineStagePaginatedResp,
  PipelineStageResp,
  PipelineStageUpdateReq,
} from '@/gen/proto/orbit/v1/pipeline/pipeline_stage';
import request from '@/utils/request';

export const pipelineStageApi = {
  list(params: {
    project_id: string;
    page?: number;
    per_page?: number;
    search?: string;
  }): Promise<PipelineStagePaginatedResp> {
    return request.get('/api/pipeline-stage', { params });
  },
  get(id: string): Promise<PipelineStageResp> {
    return request.get(`/api/pipeline-stage/${id}`);
  },
  create(data: PipelineStageCreateReq, params: { project_id: string }): Promise<PipelineStageResp> {
    return request.post('/api/pipeline-stage', data, { params });
  },
  update(id: string, data: PipelineStageUpdateReq): Promise<PipelineStageResp> {
    return request.put(`/api/pipeline-stage/${id}`, data);
  },
  delete(id: string): Promise<void> {
    return request.delete(`/api/pipeline-stage/${id}`);
  },
};
