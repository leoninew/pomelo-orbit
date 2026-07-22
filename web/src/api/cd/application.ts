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
} from '@/gen/proto/orbit/v1/application';
import type {
  ApplicationExportResp,
  ApplicationImportReq,
} from '@/gen/proto/orbit/v1/application_bundle';
import type {
  ServiceListResp,
  ServiceResp,
  VersionCreateReq,
  VersionForkReq,
  VersionListResp,
  VersionPreviewReq,
  VersionPreviewResp,
  VersionResp,
  VersionUpdateReq,
} from '@/gen/proto/orbit/v1/version';
import request from '@/utils/request';

export const applicationApi = {
  list(params?: {
    page?: number;
    per_page?: number;
    search?: string;
    project_id?: string;
  }): Promise<ApplicationPaginatedResp> {
    return request.get('/api/cd/application', { params });
  },

  get(id: string): Promise<ApplicationResp> {
    return request.get(`/api/cd/application/${id}`);
  },

  create(data: ApplicationCreateReq, params: { project_id: string }): Promise<ApplicationResp> {
    return request.post('/api/cd/application', data, { params });
  },

  update(id: string, data: ApplicationUpdateReq): Promise<ApplicationResp> {
    return request.put(`/api/cd/application/${id}`, data);
  },

  delete(id: string, removeDir: boolean = false): Promise<void> {
    return request.delete(`/api/cd/application/${id}`, {
      params: { remove_dir: removeDir },
    });
  },

  deploy(id: string, data: ApplicationDeployReq): Promise<DeploymentActionResp> {
    return request.post(`/api/cd/application/${id}/deploy`, data);
  },

  stop(id: string, data: ApplicationStopReq): Promise<DeploymentActionResp> {
    return request.post(`/api/cd/application/${id}/stop`, data);
  },

  restart(id: string, data: ApplicationRestartReq): Promise<DeploymentActionResp> {
    return request.post(`/api/cd/application/${id}/restart`, data);
  },

  getStatus(
    id: string,
    params?: { environment_id?: string; instance_key?: string; service_id?: string }
  ): Promise<ApplicationStatusResp> {
    return request.get(`/api/cd/application/${id}/status`, { params });
  },

  getLogs(
    id: string,
    params?: {
      tail?: number;
      environment_id?: string;
      instance_key?: string;
      service_id?: string;
    }
  ): Promise<ApplicationLogsResp> {
    return request.get(`/api/cd/application/${id}/logs`, { params });
  },

  exportApplication(id: string): Promise<ApplicationExportResp> {
    return request.get(`/api/cd/application/${id}/export`);
  },

  importApplication(
    data: ApplicationImportReq,
    params: { project_id: string }
  ): Promise<ApplicationResp> {
    return request.post('/api/cd/application/import', data, { params });
  },

  listServices(id: string): Promise<ServiceListResp> {
    return request.get(`/api/cd/application/${id}/service`);
  },

  getService(id: string): Promise<ServiceResp | null> {
    return applicationApi.listServices(id).then((resp) => {
      const items = resp.items ?? [];
      return items[0] ?? null;
    });
  },

  listVersions(id: string): Promise<VersionListResp> {
    return request.get(`/api/cd/application/${id}/version`);
  },

  createVersion(id: string, data: VersionCreateReq): Promise<VersionResp> {
    return request.post(`/api/cd/application/${id}/version`, data);
  },

  getVersion(versionId: string): Promise<VersionResp> {
    return request.get(`/api/cd/version/${versionId}`);
  },

  updateVersion(versionId: string, data: VersionUpdateReq): Promise<VersionResp> {
    return request.put(`/api/cd/version/${versionId}`, data);
  },

  publishVersion(versionId: string): Promise<VersionResp> {
    return request.post(`/api/cd/version/${versionId}/publish`);
  },

  forkVersion(versionId: string, data: VersionForkReq): Promise<VersionResp> {
    return request.post(`/api/cd/version/${versionId}/fork`, data);
  },

  previewVersion(versionId: string, data: VersionPreviewReq): Promise<VersionPreviewResp> {
    return request.post(`/api/cd/version/${versionId}/preview`, data);
  },
};
