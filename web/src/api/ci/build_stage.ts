import type {
  BuildStageCreateReq,
  BuildStageDuplicateReq,
  BuildStagePaginatedResp,
  BuildStageResp,
  BuildStageUpdateReq,
} from '@/gen/orbit/api/v1/build_stage';
import request from '@/utils/request';

export const buildStageApi = {
  list(params?: {
    page?: number;
    per_page?: number;
    search?: string;
    project_id?: string;
  }): Promise<BuildStagePaginatedResp> {
    return request.get('/api/ci/build-stage', { params });
  },

  get(id: string): Promise<BuildStageResp> {
    return request.get(`/api/ci/build-stage/${id}`);
  },

  create(data: BuildStageCreateReq, params: { project_id: string }): Promise<BuildStageResp> {
    return request.post('/api/ci/build-stage', data, { params });
  },

  update(id: string, data: BuildStageUpdateReq): Promise<BuildStageResp> {
    return request.put(`/api/ci/build-stage/${id}`, data);
  },

  delete(id: string): Promise<void> {
    return request.delete(`/api/ci/build-stage/${id}`);
  },

  duplicate(id: string, data: BuildStageDuplicateReq): Promise<BuildStageResp> {
    return request.post(`/api/ci/build-stage/${id}/duplicate`, data);
  },
};
