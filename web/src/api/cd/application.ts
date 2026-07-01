import type {
  Application,
  ApplicationCreateReq,
  ApplicationExportResp,
  ApplicationImportReq,
  ApplicationRoute,
  ApplicationServiceConfig,
  ApplicationUpdateReq,
  ComposeServiceResp,
  ConfigFile,
} from '@/types/cd/application';
import type { ListResp, PaginatedResp } from '@/types/common';
import request from '@/utils/request';

export interface DeploymentActionResp {
  deployment_id: string;
}

export interface ApplicationStatusResp {
  status: string;
}

export interface ApplicationLogsResp {
  logs: string;
}

export interface ApplicationFileContentResp {
  content: string;
  path: string;
}

export interface ApplicationComposePreviewResp {
  compose_yaml: string;
}

// 应用相关 API
export const applicationApi = {
  // 获取应用列表
  list(params?: {
    page?: number;
    per_page?: number;
    search?: string;
    project_id?: string;
  }): Promise<PaginatedResp<Application>> {
    return request.get('/api/cd/application', { params });
  },

  // 获取应用详情
  get(id: string): Promise<Application> {
    return request.get(`/api/cd/application/${id}`);
  },

  // 创建应用
  create(data: ApplicationCreateReq, params: { project_id: string }): Promise<Application> {
    return request.post('/api/cd/application', data, { params });
  },

  // 更新应用
  update(id: string, data: ApplicationUpdateReq): Promise<Application> {
    return request.put(`/api/cd/application/${id}`, data);
  },

  // 删除应用
  delete(id: string, removeDir: boolean = false): Promise<void> {
    return request.delete(`/api/cd/application/${id}`, {
      params: { remove_dir: removeDir },
    });
  },

  // 手动触发部署
  deploy(id: string, data?: { force_recreate?: boolean }): Promise<DeploymentActionResp> {
    return request.post(`/api/cd/application/${id}/deploy`, data ?? {});
  },

  // 停止应用
  stop(id: string, removeVolumes?: boolean): Promise<DeploymentActionResp> {
    return request.post(`/api/cd/application/${id}/stop`, {
      remove_volumes: removeVolumes,
    });
  },

  // 重启应用
  restart(id: string): Promise<DeploymentActionResp> {
    return request.post(`/api/cd/application/${id}/restart`);
  },

  // 获取应用状态
  getStatus(id: string): Promise<ApplicationStatusResp> {
    return request.get(`/api/cd/application/${id}/status`);
  },

  // 获取应用日志
  getLogs(id: string, tail?: number): Promise<ApplicationLogsResp> {
    return request.get(`/api/cd/application/${id}/logs`, { params: { tail } });
  },

  // 读取应用文件
  readFile(id: string, fileId: string): Promise<ApplicationFileContentResp> {
    return request.get(`/api/cd/application/${id}/file/${fileId}`);
  },

  // 写入应用文件
  writeFile(id: string, fileId: string, path: string, content: string): Promise<ConfigFile> {
    return request.put(`/api/cd/application/${id}/file/${fileId}`, {
      path,
      content,
    });
  },

  // 创建应用文件
  createFile(id: string, path: string, content: string = ''): Promise<ConfigFile> {
    return request.post(`/api/cd/application/${id}/file`, { path, content });
  },

  // 获取应用文件列表
  listFiles(id: string): Promise<ListResp<ConfigFile>> {
    return request.get(`/api/cd/application/${id}/files`);
  },

  /** 预览部署时生成的 docker-compose.yml（模板渲染、镜像覆盖、路由 labels） */
  previewCompose(id: string): Promise<ApplicationComposePreviewResp> {
    return request.post(`/api/cd/application/${id}/compose-preview`);
  },

  // 删除应用文件
  deleteFile(id: string, fileId: string): Promise<void> {
    return request.delete(`/api/cd/application/${id}/file/${fileId}`);
  },

  // 导出应用
  exportApplication(id: string): Promise<ApplicationExportResp> {
    return request.get(`/api/cd/application/${id}/export`);
  },

  // 导入应用
  importApplication(
    data: ApplicationImportReq,
    params: { project_id: string }
  ): Promise<Application> {
    return request.post('/api/cd/application/import', data, { params });
  },

  // 获取路由托管列表
  listRoutes(id: string): Promise<ListResp<ApplicationRoute>> {
    return request.get(`/api/cd/application/${id}/route`);
  },

  // 创建路由托管
  createRoute(
    id: string,
    data: { service_name: string; domain: string; port: number }
  ): Promise<ApplicationRoute> {
    return request.post(`/api/cd/application/${id}/route`, data);
  },

  // 更新路由托管
  updateRoute(
    id: string,
    routeId: string,
    data: { service_name: string; domain: string; port: number }
  ): Promise<ApplicationRoute> {
    return request.put(`/api/cd/application/${id}/route/${routeId}`, data);
  },

  // 删除路由托管
  deleteRoute(id: string, routeId: string): Promise<void> {
    return request.delete(`/api/cd/application/${id}/route/${routeId}`);
  },

  // 解析 docker-compose service 列表
  listComposeServices(id: string): Promise<ListResp<ComposeServiceResp>> {
    return request.get(`/api/cd/application/${id}/compose-service`);
  },

  // 获取 service 级配置
  listServiceConfigs(id: string): Promise<ListResp<ApplicationServiceConfig>> {
    return request.get(`/api/cd/application/${id}/service-config`);
  },

  // 更新 service 级配置
  updateServiceConfig(
    id: string,
    serviceName: string,
    image: string | null
  ): Promise<ApplicationServiceConfig> {
    return request.put(`/api/cd/application/${id}/service-config/${serviceName}`, { image });
  },
};
