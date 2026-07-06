import type {
  DeploymentCancelReq,
  DeploymentContainerLogsResp,
  DeploymentLogsResp,
  DeploymentPaginatedResp,
  DeploymentResp,
} from '@/gen/orbit/api/v1/deployment';
import request from '@/utils/request';

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
    return request.get('/api/cd/deployment', { params });
  },

  // 获取部署记录详情
  get(id: string): Promise<DeploymentResp> {
    return request.get(`/api/cd/deployment/${id}`);
  },

  // 取消部署
  cancel(id: string, data: DeploymentCancelReq): Promise<DeploymentResp> {
    return request.post(`/api/cd/deployment/${id}/cancel`, data);
  },

  // 获取部署日志（增量读取，回退用）
  getLogs(id: string, offset: number = 0): Promise<DeploymentLogsResp> {
    return request.get(`/api/cd/deployment/${id}/logs`, {
      params: { offset },
    });
  },

  // 获取部署对应的容器日志
  getContainerLogs(id: string, params?: { tail?: number }): Promise<DeploymentContainerLogsResp> {
    return request.get(`/api/cd/deployment/${id}/container-logs`, { params });
  },
};
