import type {
  ProjectInitializationBootstrapReq,
  ProjectInitializationEnvironmentReq,
  ProjectInitializationGatewayReq,
  ProjectInitializationStatusResp,
  ProjectInitializationWindowsCommandResp,
} from '@/gen/proto/orbit/v1/project_initialization/project_initialization';
import request, { remoteRequestConfig } from '@/utils/request';

export const projectInitializationApi = {
  getStatus(projectId: string): Promise<ProjectInitializationStatusResp> {
    return request.get('/api/project-initialization', { params: { project_id: projectId } });
  },

  testEnvironment(
    projectId: string,
    data: ProjectInitializationEnvironmentReq
  ): Promise<{ ok: boolean }> {
    return request.post(
      '/api/project-initialization/environment/test',
      data,
      remoteRequestConfig({ params: { project_id: projectId } })
    );
  },

  saveEnvironment(
    projectId: string,
    data: ProjectInitializationEnvironmentReq
  ): Promise<ProjectInitializationStatusResp> {
    return request.post('/api/project-initialization/environment', data, {
      params: { project_id: projectId },
    });
  },

  prepareWindowsEnvironment(
    projectId: string,
    data: ProjectInitializationEnvironmentReq
  ): Promise<ProjectInitializationWindowsCommandResp> {
    return request.post('/api/project-initialization/environment/windows-command', data, {
      params: { project_id: projectId },
    });
  },

  bootstrapEnvironment(
    projectId: string,
    data: ProjectInitializationBootstrapReq
  ): Promise<ProjectInitializationStatusResp> {
    return request.post(
      '/api/project-initialization/bootstrap',
      data,
      remoteRequestConfig({ params: { project_id: projectId } })
    );
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
