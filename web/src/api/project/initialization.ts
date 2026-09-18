import type {
  ProjectInitializationEnvironmentReq,
  ProjectInitializationGatewayReq,
  ProjectInitializationSSHCommandResp,
  ProjectInitializationStatusResp,
} from '@/gen/proto/orbit/v1/project_initialization/project_initialization';
import request, { remoteRequestConfig } from '@/utils/request';

export const projectInitializationApi = {
  getStatus(projectId: string): Promise<ProjectInitializationStatusResp> {
    return request.get('/api/project-initialization', { params: { project_id: projectId } });
  },

  saveEnvironment(
    projectId: string,
    data: ProjectInitializationEnvironmentReq
  ): Promise<ProjectInitializationStatusResp> {
    return request.post('/api/project-initialization/environment', data, {
      params: { project_id: projectId },
    });
  },

  prepareSSHEnvironment(
    projectId: string,
    data: ProjectInitializationEnvironmentReq
  ): Promise<ProjectInitializationSSHCommandResp> {
    return request.post('/api/project-initialization/environment/ssh-command', data, {
      params: { project_id: projectId },
    });
  },

  probeEnvironment(projectId: string): Promise<ProjectInitializationStatusResp> {
    return request.post(
      '/api/project-initialization/probe',
      undefined,
      remoteRequestConfig({ params: { project_id: projectId } })
    );
  },

  createGateway(
    projectId: string,
    data: ProjectInitializationGatewayReq
  ): Promise<ProjectInitializationStatusResp> {
    return request.post('/api/project-initialization/gateway', data, {
      params: { project_id: projectId },
    });
  },
};
