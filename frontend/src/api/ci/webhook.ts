import type {
  ProjectWebhookCreateReq,
  ProjectWebhookUpdateReq,
  RepositoryWebhook,
} from '@/types/ci/webhook';
import request from '@/utils/request';

export const webhookApi = {
  list(repositoryId: string, params: { project_id: string }): Promise<RepositoryWebhook[]> {
    return request.get(`/api/ci/repository/${repositoryId}/webhook`, { params });
  },

  get(webhookId: string): Promise<RepositoryWebhook> {
    return request.get(`/api/ci/webhook/${webhookId}`);
  },

  create(repositoryId: string, data: ProjectWebhookCreateReq): Promise<RepositoryWebhook> {
    return request.post(`/api/ci/repository/${repositoryId}/webhook`, data);
  },

  update(
    repositoryId: string,
    webhookId: string,
    data: ProjectWebhookUpdateReq
  ): Promise<RepositoryWebhook> {
    return request.put(`/api/ci/repository/${repositoryId}/webhook/${webhookId}`, data);
  },

  delete(repositoryId: string, webhookId: string): Promise<void> {
    return request.delete(`/api/ci/repository/${repositoryId}/webhook/${webhookId}`);
  },
};
