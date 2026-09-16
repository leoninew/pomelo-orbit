import type {
  ApplicationCreateReq,
  ApplicationLogsResp,
  ApplicationPaginatedResp,
  ApplicationResp,
  ApplicationRestartReq,
  ApplicationStatusResp,
  ApplicationStopReq,
  ApplicationUpdateReq,
  DeploymentActionResp,
} from '@/gen/proto/orbit/v1/application/application';
import type {
  VersionComponentAdvancedUpdateReq,
  VersionComponentBasicUpdateReq,
  VersionComponentCreateReq,
  VersionComponentDependenciesUpdateReq,
  VersionComponentDevicesUpdateReq,
  VersionComponentEnvUpdateReq,
  VersionComponentMountsUpdateReq,
  VersionComponentEndpointsUpdateReq,
  VersionComponentResp,
  VersionComponentRuntimeUpdateReq,
  VersionCreateReq,
  VersionForkReq,
  VersionPaginatedResp,
  VersionPreviewReq,
  VersionPreviewResp,
  VersionResp,
  VersionUpdateReq,
} from '@/gen/proto/orbit/v1/application/version';
import type { ServiceListResp } from '@/gen/proto/orbit/v1/service/service';
import request, { remoteRequestConfig, type AxiosRequestConfig } from '@/utils/request';

export const applicationApi = {
  list(
    projectId: string,
    params?: {
      page?: number;
      per_page?: number;
      search?: string;
      kind?: string;
    }
  ): Promise<ApplicationPaginatedResp> {
    return request.get('/api/application', { params: { project_id: projectId, ...params } });
  },

  get(projectId: string, id: string): Promise<ApplicationResp> {
    return request.get(`/api/application/${id}`, { params: { project_id: projectId } });
  },

  create(projectId: string, data: ApplicationCreateReq): Promise<ApplicationResp> {
    return request.post('/api/application', data, { params: { project_id: projectId } });
  },

  update(projectId: string, id: string, data: ApplicationUpdateReq): Promise<ApplicationResp> {
    return request.put(`/api/application/${id}`, data, { params: { project_id: projectId } });
  },

  delete(projectId: string, id: string): Promise<void> {
    return request.delete(`/api/application/${id}`, { params: { project_id: projectId } });
  },

  stop(projectId: string, id: string, data: ApplicationStopReq): Promise<DeploymentActionResp> {
    return request.post(`/api/application/${id}/stop`, data, { params: { project_id: projectId } });
  },

  restart(
    projectId: string,
    id: string,
    data: ApplicationRestartReq
  ): Promise<DeploymentActionResp> {
    return request.post(`/api/application/${id}/restart`, data, {
      params: { project_id: projectId },
    });
  },

  getStatus(
    projectId: string,
    id: string,
    params?: { service_id?: string }
  ): Promise<ApplicationStatusResp> {
    return request.get(
      `/api/application/${id}/status`,
      remoteRequestConfig({ params: { project_id: projectId, ...params } })
    );
  },

  getLogs(
    projectId: string,
    id: string,
    params?: {
      tail?: number;
      service_id?: string;
      component?: string;
    },
    config?: AxiosRequestConfig
  ): Promise<ApplicationLogsResp> {
    return request.get(
      `/api/application/${id}/logs`,
      remoteRequestConfig({ ...config, params: { project_id: projectId, ...params } })
    );
  },

  listServices(projectId: string, id: string): Promise<ServiceListResp> {
    return request.get(`/api/application/${id}/service`, { params: { project_id: projectId } });
  },

  listVersions(
    projectId: string,
    id: string,
    params?: { page?: number; per_page?: number; search?: string }
  ): Promise<VersionPaginatedResp> {
    return request.get(`/api/application/${id}/version`, {
      params: { project_id: projectId, ...params },
    });
  },

  createVersion(projectId: string, id: string, data: VersionCreateReq): Promise<VersionResp> {
    return request.post(`/api/application/${id}/version`, data, {
      params: { project_id: projectId },
    });
  },

  getVersion(projectId: string, versionId: string): Promise<VersionResp> {
    return request.get(`/api/version/${versionId}`, { params: { project_id: projectId } });
  },

  previewVersion(
    projectId: string,
    versionId: string,
    data: VersionPreviewReq
  ): Promise<VersionPreviewResp> {
    return request.post(`/api/version/${versionId}/preview`, data, {
      params: { project_id: projectId },
    });
  },

  getVersionComponent(
    projectId: string,
    versionId: string,
    componentId: string
  ): Promise<VersionComponentResp> {
    return request.get(`/api/version/${versionId}/component/${componentId}`, {
      params: { project_id: projectId },
    });
  },

  createVersionComponent(
    projectId: string,
    versionId: string,
    data: VersionComponentCreateReq
  ): Promise<VersionComponentResp> {
    return request.post(`/api/version/${versionId}/component`, data, {
      params: { project_id: projectId },
    });
  },

  updateVersionComponentBasic(
    projectId: string,
    versionId: string,
    componentId: string,
    data: VersionComponentBasicUpdateReq
  ): Promise<VersionComponentResp> {
    return request.put(`/api/version/${versionId}/component/${componentId}/basic`, data, {
      params: { project_id: projectId },
    });
  },

  updateVersionComponentRuntime(
    projectId: string,
    versionId: string,
    componentId: string,
    data: VersionComponentRuntimeUpdateReq
  ): Promise<VersionComponentResp> {
    return request.put(`/api/version/${versionId}/component/${componentId}/runtime`, data, {
      params: { project_id: projectId },
    });
  },

  updateVersionComponentEndpoints(
    projectId: string,
    versionId: string,
    componentId: string,
    data: VersionComponentEndpointsUpdateReq
  ): Promise<VersionComponentResp> {
    return request.put(`/api/version/${versionId}/component/${componentId}/endpoints`, data, {
      params: { project_id: projectId },
    });
  },

  updateVersionComponentEnv(
    projectId: string,
    versionId: string,
    componentId: string,
    data: VersionComponentEnvUpdateReq
  ): Promise<VersionComponentResp> {
    return request.put(`/api/version/${versionId}/component/${componentId}/env`, data, {
      params: { project_id: projectId },
    });
  },

  updateVersionComponentMounts(
    projectId: string,
    versionId: string,
    componentId: string,
    data: VersionComponentMountsUpdateReq
  ): Promise<VersionComponentResp> {
    return request.put(`/api/version/${versionId}/component/${componentId}/mounts`, data, {
      params: { project_id: projectId },
    });
  },

  updateVersionComponentDependencies(
    projectId: string,
    versionId: string,
    componentId: string,
    data: VersionComponentDependenciesUpdateReq
  ): Promise<VersionComponentResp> {
    return request.put(`/api/version/${versionId}/component/${componentId}/dependencies`, data, {
      params: { project_id: projectId },
    });
  },

  updateVersionComponentDevices(
    projectId: string,
    versionId: string,
    componentId: string,
    data: VersionComponentDevicesUpdateReq
  ): Promise<VersionComponentResp> {
    return request.put(`/api/version/${versionId}/component/${componentId}/devices`, data, {
      params: { project_id: projectId },
    });
  },

  updateVersionComponentAdvanced(
    projectId: string,
    versionId: string,
    componentId: string,
    data: VersionComponentAdvancedUpdateReq
  ): Promise<VersionComponentResp> {
    return request.put(`/api/version/${versionId}/component/${componentId}/advanced`, data, {
      params: { project_id: projectId },
    });
  },

  deleteVersionComponent(projectId: string, versionId: string, componentId: string): Promise<void> {
    return request.delete(`/api/version/${versionId}/component/${componentId}`, {
      params: { project_id: projectId },
    });
  },

  updateVersion(
    projectId: string,
    versionId: string,
    data: VersionUpdateReq
  ): Promise<VersionResp> {
    return request.put(`/api/version/${versionId}`, data, { params: { project_id: projectId } });
  },

  publishVersion(projectId: string, versionId: string): Promise<VersionResp> {
    return request.post(`/api/version/${versionId}/publish`, undefined, {
      params: { project_id: projectId },
    });
  },

  unpublishVersion(projectId: string, versionId: string): Promise<VersionResp> {
    return request.post(`/api/version/${versionId}/unpublish`, undefined, {
      params: { project_id: projectId },
    });
  },

  deleteVersion(projectId: string, versionId: string): Promise<void> {
    return request.delete(`/api/version/${versionId}`, { params: { project_id: projectId } });
  },

  forkVersion(projectId: string, versionId: string, data: VersionForkReq): Promise<VersionResp> {
    return request.post(`/api/version/${versionId}/fork`, data, {
      params: { project_id: projectId },
    });
  },
};
