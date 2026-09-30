import type {
  DeploymentCancelReq,
  DeploymentPaginatedResp,
  DeploymentResp,
} from '@/gen/proto/orbit/v1/deployment/deployment';
import request, { type AxiosRequestConfig } from '@/utils/request';

// 部署记录相关 API
export const deploymentApi = {
  // 获取部署记录列表
  list(
    projectId: string,
    params?: {
      page?: number;
      per_page?: number;
      application_id?: string;
      status?: string;
      search?: string;
      date_from?: string;
      date_to?: string;
    }
  ): Promise<DeploymentPaginatedResp> {
    return request.get('/api/deployment', { params: { project_id: projectId, ...params } });
  },

  // 获取部署记录详情
  get(projectId: string, id: string, config?: AxiosRequestConfig): Promise<DeploymentResp> {
    return request.get(`/api/deployment/${id}`, {
      ...config,
      params: { project_id: projectId, ...config?.params },
    });
  },

  // 删除已完成的部署记录及日志文件
  delete(projectId: string, id: string): Promise<void> {
    return request.delete(`/api/deployment/${id}`, { params: { project_id: projectId } });
  },

  // 取消部署
  cancel(projectId: string, id: string, data: DeploymentCancelReq): Promise<DeploymentResp> {
    return request.post(`/api/deployment/${id}/cancel`, data, {
      params: { project_id: projectId },
    });
  },
};
