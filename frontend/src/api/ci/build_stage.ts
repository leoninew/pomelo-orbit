import type { BuildStage, BuildStageCreateReq, BuildStageUpdateReq } from '@/types/ci/template';
import type { PaginatedResp } from '@/types/common';
import request from '@/utils/request';

export const buildStageApi = {
	list(params?: {
		page?: number
		per_page?: number
		search?: string
		projectId?: string
	}): Promise<PaginatedResp<BuildStage>> {
		return request.get('/api/ci/build-stage', { params });
	},

	get(id: string, params: { projectId: string }): Promise<BuildStage> {
		return request.get(`/api/ci/build-stage/${id}`, { params });
	},

	create(data: BuildStageCreateReq, params: { projectId: string }): Promise<BuildStage> {
		return request.post('/api/ci/build-stage', data, { params });
	},

	update(id: string, data: BuildStageUpdateReq): Promise<BuildStage> {
		return request.put(`/api/ci/build-stage/${id}`, data);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/ci/build-stage/${id}`);
	},

	duplicate(id: string, params: { projectId: string }): Promise<BuildStage> {
		return request.post(`/api/ci/build-stage/${id}/duplicate`, {}, { params });
	},
};
