import type {
	ProjectWebhookCreateReq,
	ProjectWebhookUpdateReq,
	RepositoryWebhook,
} from '@/types/api';
import request from '@/utils/request';

export const webhookApi = {
	list(projectId: string): Promise<RepositoryWebhook[]> {
		return request.get(`/api/ci/repository/${projectId}/webhook`);
	},

	get(webhookId: string): Promise<RepositoryWebhook> {
		return request.get(`/api/ci/webhook/${webhookId}`);
	},

	create(projectId: string, data: ProjectWebhookCreateReq): Promise<RepositoryWebhook> {
		return request.post(`/api/ci/repository/${projectId}/webhook`, data);
	},

	update(
		projectId: string,
		webhookId: string,
		data: ProjectWebhookUpdateReq
	): Promise<RepositoryWebhook> {
		return request.put(`/api/ci/repository/${projectId}/webhook/${webhookId}`, data);
	},

	delete(projectId: string, webhookId: string): Promise<void> {
		return request.delete(`/api/ci/repository/${projectId}/webhook/${webhookId}`);
	},
};
