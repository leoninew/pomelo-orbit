import type { Deployment, DeploymentDetail } from '@/types/cd/deployment';
import type { PaginatedResp } from '@/types/common';
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
  }): Promise<PaginatedResp<Deployment>> {
    return request.get('/api/cd/deployment', { params });
  },

  // 获取部署记录详情
  get(id: string): Promise<DeploymentDetail> {
    return request.get(`/api/cd/deployment/${id}`);
  },

  // 取消部署
  cancel(id: string): Promise<void> {
    return request.post(`/api/cd/deployment/${id}/cancel`);
  },

  // 获取部署日志（增量读取，回退用）
  getLogs(
    id: string,
    offset: number = 0
  ): Promise<{
    logs: string;
    offset: number;
    is_complete: boolean;
    status: string;
  }> {
    return request.get(`/api/cd/deployment/${id}/logs`, {
      params: { offset },
    });
  },

  // 获取部署对应的容器日志
  getContainerLogs(
    id: string,
    params?: { tail?: number }
  ): Promise<{
    logs: string;
    source: 'since' | 'tail';
    is_realtime_supported: boolean;
  }> {
    return request.get(`/api/cd/deployment/${id}/container-logs`, { params });
  },

  // SSE 流式获取部署日志
  streamLogs(id: string, token: string | null, signal?: AbortSignal): Promise<Response> {
    return fetch(`/api/cd/deployment/${id}/stream-log`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      signal,
    });
  },
};
