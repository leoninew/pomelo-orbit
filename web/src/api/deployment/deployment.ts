import type {
  DeploymentCancelReq,
  DeploymentContainerLogsResp,
  DeploymentLogsResp,
  DeploymentPaginatedResp,
  DeploymentResp,
} from '@/gen/proto/orbit/v1/deployment/deployment';
import request, { type AxiosRequestConfig } from '@/utils/request';

// 部署记录相关 API
export const deploymentApi = {
  // 获取部署记录列表
  list(params?: {
    page?: number;
    per_page?: number;
    application_id?: string;
    status?: string;
    search?: string;
    date_from?: string;
    date_to?: string;
    project_id?: string;
  }): Promise<DeploymentPaginatedResp> {
    return request.get('/api/deployment', { params });
  },

  // 获取部署记录详情
  get(id: string, config?: AxiosRequestConfig): Promise<DeploymentResp> {
    return request.get(`/api/deployment/${id}`, config);
  },

  // 删除已完成的部署记录及日志文件
  delete(id: string): Promise<void> {
    return request.delete(`/api/deployment/${id}`);
  },

  // 取消部署
  cancel(id: string, data: DeploymentCancelReq): Promise<DeploymentResp> {
    return request.post(`/api/deployment/${id}/cancel`, data);
  },

  // 获取部署日志（增量读取；失败详情主展示）
  getLogs(
    id: string,
    offset: number = 0,
    config?: AxiosRequestConfig
  ): Promise<DeploymentLogsResp> {
    return request.get(`/api/deployment/${id}/logs`, {
      ...config,
      params: { offset },
    });
  },

  // 获取部署对应的容器日志
  getContainerLogs(
    id: string,
    params?: { tail?: number },
    config?: AxiosRequestConfig
  ): Promise<DeploymentContainerLogsResp> {
    return request.get(`/api/deployment/${id}/container-logs`, { ...config, params });
  },
};
