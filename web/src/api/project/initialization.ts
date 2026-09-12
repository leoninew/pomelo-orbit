import type {
  ProjectInitializationBootstrapReq,
  ProjectInitializationEnvironmentReq,
  ProjectInitializationGatewayReq,
  ProjectInitializationStatusResp,
} from '@/gen/proto/orbit/v1/project_initialization/project_initialization';
import request from '@/utils/request';

export const projectInitializationApi = {
  getStatus(projectId: string): Promise<ProjectInitializationStatusResp> {
    return request.get(`/api/project/${projectId}/initialization`);
  },

  testEnvironment(
    projectId: string,
    data: ProjectInitializationEnvironmentReq
  ): Promise<{ ok: boolean }> {
    return request.post(`/api/project/${projectId}/initialization/environment/test`, data);
  },

  saveEnvironment(
    projectId: string,
    data: ProjectInitializationEnvironmentReq
  ): Promise<ProjectInitializationStatusResp> {
    return request.post(`/api/project/${projectId}/initialization/environment`, data);
  },

  getDeploymentPublicKey(projectId: string): Promise<{ public_key: string }> {
    return request.get(`/api/project/${projectId}/initialization/environment/deployment-key`);
  },

  bootstrapEnvironment(
    projectId: string,
    data: ProjectInitializationBootstrapReq
  ): Promise<ProjectInitializationStatusResp> {
    return request.post(`/api/project/${projectId}/initialization/bootstrap`, data);
  },

  probeEnvironment(projectId: string): Promise<ProjectInitializationStatusResp> {
    return request.post(`/api/project/${projectId}/initialization/probe`);
  },

  createGateway(
    projectId: string,
    data: ProjectInitializationGatewayReq
  ): Promise<ProjectInitializationStatusResp> {
    return request.post(`/api/project/${projectId}/initialization/gateway`, data);
  },
};
