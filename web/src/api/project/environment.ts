import type {
  EnvironmentResp,
  ProjectEnvironmentInitializeReq,
  ProjectEnvironmentUpdateReq,
} from '@/gen/proto/orbit/v1/environment/environment';
import request, { remoteRequestConfig } from '@/utils/request';

export const projectEnvironmentApi = {
  get(projectId: string): Promise<EnvironmentResp> {
    return request.get(`/api/project/${projectId}/environment`);
  },

  update(projectId: string, data: ProjectEnvironmentUpdateReq): Promise<EnvironmentResp> {
    return request.put(`/api/project/${projectId}/environment`, data);
  },

  probe(projectId: string): Promise<EnvironmentResp> {
    return request.post(
      `/api/project/${projectId}/environment/probe`,
      undefined,
      remoteRequestConfig()
    );
  },

  initialize(projectId: string, data: ProjectEnvironmentInitializeReq): Promise<EnvironmentResp> {
    return request.post(
      `/api/project/${projectId}/environment/initialize`,
      data,
      remoteRequestConfig()
    );
  },
};
