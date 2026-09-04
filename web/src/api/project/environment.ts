import type {
  EnvironmentResp,
  ProjectEnvironmentUpdateReq,
} from '@/gen/proto/orbit/v1/environment/environment';
import request from '@/utils/request';

export const projectEnvironmentApi = {
  get(projectId: string): Promise<EnvironmentResp> {
    return request.get(`/api/project/${projectId}/environment`);
  },

  update(projectId: string, data: ProjectEnvironmentUpdateReq): Promise<EnvironmentResp> {
    return request.put(`/api/project/${projectId}/environment`, data);
  },

  probe(projectId: string): Promise<EnvironmentResp> {
    return request.post(`/api/project/${projectId}/environment/probe`);
  },
};
