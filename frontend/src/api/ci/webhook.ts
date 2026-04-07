import type { ProjectWebhook, ProjectWebhookCreateReq, ProjectWebhookUpdateReq } from '@/types/api';
import request from '@/utils/request';

export const webhookApi = {
	list(projectId: string): Promise<ProjectWebhook[]> {
		return request.get(`/api/ci/projects/${projectId}/webhooks`);
	},

	get(webhookId: string): Promise<ProjectWebhook> {
		return request.get(`/api/ci/webhooks/${webhookId}`);
	},

	create(projectId: string, data: ProjectWebhookCreateReq): Promise<ProjectWebhook> {
		return request.post(`/api/ci/projects/${projectId}/webhooks`, data);
	},

	update(
		projectId: string,
		webhookId: string,
		data: ProjectWebhookUpdateReq
	): Promise<ProjectWebhook> {
		return request.put(`/api/ci/projects/${projectId}/webhooks/${webhookId}`, data);
	},

	delete(projectId: string, webhookId: string): Promise<void> {
		return request.delete(`/api/ci/projects/${projectId}/webhooks/${webhookId}`);
	},
};
