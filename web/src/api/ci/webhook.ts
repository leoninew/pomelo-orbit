import type {
  RepositoryWebhookCreateReq,
  RepositoryWebhookListResp,
  RepositoryWebhookResp,
  RepositoryWebhookUpdateReq,
} from '@/gen/orbit/api/v1/webhook';
import request from '@/utils/request';

export const webhookApi = {
  list(repositoryId: string, params: { project_id: string }): Promise<RepositoryWebhookListResp> {
    return request.get(`/api/ci/repository/${repositoryId}/webhook`, { params });
  },

  get(webhookId: string): Promise<RepositoryWebhookResp> {
    return request.get(`/api/ci/webhook/${webhookId}`);
  },

  create(repositoryId: string, data: RepositoryWebhookCreateReq): Promise<RepositoryWebhookResp> {
    return request.post(`/api/ci/repository/${repositoryId}/webhook`, data);
  },

  update(
    repositoryId: string,
    webhookId: string,
    data: RepositoryWebhookUpdateReq
  ): Promise<RepositoryWebhookResp> {
    return request.put(`/api/ci/repository/${repositoryId}/webhook/${webhookId}`, data);
  },

  delete(repositoryId: string, webhookId: string): Promise<void> {
    return request.delete(`/api/ci/repository/${repositoryId}/webhook/${webhookId}`);
  },
};
