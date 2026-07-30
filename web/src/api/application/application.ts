import type {
  ApplicationCreateReq,
  ApplicationDeployReq,
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
  ApplicationExportResp,
  ApplicationImportReq,
} from '@/gen/proto/orbit/v1/application/application_bundle';
import type {
  VersionComponentAdvancedUpdateReq,
  VersionComponentBasicUpdateReq,
  VersionComponentCreateReq,
  VersionComponentDependenciesUpdateReq,
  VersionComponentEnvUpdateReq,
  VersionComponentMountsUpdateReq,
  VersionComponentPortsUpdateReq,
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
import request from '@/utils/request';

export const applicationApi = {
  list(params?: {
    page?: number;
    per_page?: number;
    search?: string;
    project_id?: string;
    kind?: string;
  }): Promise<ApplicationPaginatedResp> {
    return request.get('/api/application', { params });
  },

  get(id: string): Promise<ApplicationResp> {
    return request.get(`/api/application/${id}`);
  },

  create(data: ApplicationCreateReq, params: { project_id: string }): Promise<ApplicationResp> {
    return request.post('/api/application', data, { params });
  },

  update(id: string, data: ApplicationUpdateReq): Promise<ApplicationResp> {
    return request.put(`/api/application/${id}`, data);
  },

  delete(id: string, removeDir: boolean = false): Promise<void> {
    return request.delete(`/api/application/${id}`, {
      params: { remove_dir: removeDir },
    });
  },

  deploy(id: string, data: ApplicationDeployReq): Promise<DeploymentActionResp> {
    return request.post(`/api/application/${id}/deploy`, data);
  },

  stop(id: string, data: ApplicationStopReq): Promise<DeploymentActionResp> {
    return request.post(`/api/application/${id}/stop`, data);
  },

  restart(id: string, data: ApplicationRestartReq): Promise<DeploymentActionResp> {
    return request.post(`/api/application/${id}/restart`, data);
  },

  getStatus(id: string, params?: { service_id?: string }): Promise<ApplicationStatusResp> {
    return request.get(`/api/application/${id}/status`, { params });
  },

  getLogs(
    id: string,
    params?: {
      tail?: number;
      service_id?: string;
      component?: string;
    }
  ): Promise<ApplicationLogsResp> {
    return request.get(`/api/application/${id}/logs`, { params });
  },

  exportApplication(id: string): Promise<ApplicationExportResp> {
    return request.get(`/api/application/${id}/export`);
  },

  importApplication(
    data: ApplicationImportReq,
    params: { project_id: string }
  ): Promise<ApplicationResp> {
    return request.post('/api/application/import', data, { params });
  },

  listServices(id: string): Promise<ServiceListResp> {
    return request.get(`/api/application/${id}/service`);
  },

  listVersions(
    id: string,
    params?: { page?: number; per_page?: number; search?: string }
  ): Promise<VersionPaginatedResp> {
    return request.get(`/api/application/${id}/version`, { params });
  },

  createVersion(id: string, data: VersionCreateReq): Promise<VersionResp> {
    return request.post(`/api/application/${id}/version`, data);
  },

  getVersion(versionId: string): Promise<VersionResp> {
    return request.get(`/api/version/${versionId}`);
  },

  getVersionComponent(versionId: string, componentId: string): Promise<VersionComponentResp> {
    return request.get(`/api/version/${versionId}/component/${componentId}`);
  },

  createVersionComponent(
    versionId: string,
    data: VersionComponentCreateReq
  ): Promise<VersionComponentResp> {
    return request.post(`/api/version/${versionId}/component`, data);
  },

  updateVersionComponentBasic(
    versionId: string,
    componentId: string,
    data: VersionComponentBasicUpdateReq
  ): Promise<VersionComponentResp> {
    return request.put(`/api/version/${versionId}/component/${componentId}/basic`, data);
  },

  updateVersionComponentRuntime(
    versionId: string,
    componentId: string,
    data: VersionComponentRuntimeUpdateReq
  ): Promise<VersionComponentResp> {
    return request.put(`/api/version/${versionId}/component/${componentId}/runtime`, data);
  },

  updateVersionComponentPorts(
    versionId: string,
    componentId: string,
    data: VersionComponentPortsUpdateReq
  ): Promise<VersionComponentResp> {
    return request.put(`/api/version/${versionId}/component/${componentId}/ports`, data);
  },

  updateVersionComponentEnv(
    versionId: string,
    componentId: string,
    data: VersionComponentEnvUpdateReq
  ): Promise<VersionComponentResp> {
    return request.put(`/api/version/${versionId}/component/${componentId}/env`, data);
  },

  updateVersionComponentMounts(
    versionId: string,
    componentId: string,
    data: VersionComponentMountsUpdateReq
  ): Promise<VersionComponentResp> {
    return request.put(`/api/version/${versionId}/component/${componentId}/mounts`, data);
  },

  updateVersionComponentDependencies(
    versionId: string,
    componentId: string,
    data: VersionComponentDependenciesUpdateReq
  ): Promise<VersionComponentResp> {
    return request.put(`/api/version/${versionId}/component/${componentId}/dependencies`, data);
  },

  updateVersionComponentAdvanced(
    versionId: string,
    componentId: string,
    data: VersionComponentAdvancedUpdateReq
  ): Promise<VersionComponentResp> {
    return request.put(`/api/version/${versionId}/component/${componentId}/advanced`, data);
  },

  deleteVersionComponent(versionId: string, componentId: string): Promise<void> {
    return request.delete(`/api/version/${versionId}/component/${componentId}`);
  },

  updateVersion(versionId: string, data: VersionUpdateReq): Promise<VersionResp> {
    return request.put(`/api/version/${versionId}`, data);
  },

  publishVersion(versionId: string): Promise<VersionResp> {
    return request.post(`/api/version/${versionId}/publish`);
  },

  unpublishVersion(versionId: string): Promise<VersionResp> {
    return request.post(`/api/version/${versionId}/unpublish`);
  },

  deleteVersion(versionId: string): Promise<void> {
    return request.delete(`/api/version/${versionId}`);
  },

  forkVersion(versionId: string, data: VersionForkReq): Promise<VersionResp> {
    return request.post(`/api/version/${versionId}/fork`, data);
  },

  previewVersion(versionId: string, data: VersionPreviewReq): Promise<VersionPreviewResp> {
    return request.post(`/api/version/${versionId}/preview`, data);
  },
};
