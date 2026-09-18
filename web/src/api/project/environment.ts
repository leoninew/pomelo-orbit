import type {
  EnvironmentResp,
  ProjectEnvironmentSSHCommandResp,
  ProjectEnvironmentUpdateReq,
} from '@/gen/proto/orbit/v1/environment/environment';
import request, { remoteRequestConfig } from '@/utils/request';

export const projectEnvironmentApi = {
  get(projectId: string): Promise<EnvironmentResp> {
    return request.get('/api/environment', { params: { project_id: projectId } });
  },

  update(projectId: string, data: ProjectEnvironmentUpdateReq): Promise<EnvironmentResp> {
    return request.put('/api/environment', data, { params: { project_id: projectId } });
  },

  probe(projectId: string): Promise<EnvironmentResp> {
    return request.post(
      '/api/environment/probe',
      undefined,
      remoteRequestConfig({ params: { project_id: projectId } })
    );
  },

  prepareSSHCommand(
    projectId: string,
    data: ProjectEnvironmentUpdateReq
  ): Promise<ProjectEnvironmentSSHCommandResp> {
    return request.post('/api/environment/ssh-command', data, {
      params: { project_id: projectId },
    });
  },
};
