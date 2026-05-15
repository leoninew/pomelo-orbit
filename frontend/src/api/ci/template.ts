import type { PaginatedResp } from '@/types/common';
import type { PipelineSnapshot } from '@/types/ci/snapshot';
import type {
	PipelineTemplate,
	PipelineTemplateCreateReq,
	StageOrchestration,
	PipelineTemplateUpdateReq,
	VariableDeclaration,
} from '@/types/ci/template';
import request from '@/utils/request';

export const pipelineTemplateApi = {
	list(params?: {
		page?: number
		per_page?: number
		search?: string
		projectId?: string
	}): Promise<PaginatedResp<PipelineTemplate>> {
		return request.get('/api/ci/template', { params });
	},

	get(id: string, params: { projectId: string }): Promise<PipelineTemplate> {
		return request.get(`/api/ci/template/${id}`, { params });
	},

	create(
		data: PipelineTemplateCreateReq,
		params: { projectId: string }
	): Promise<PipelineTemplate> {
		return request.post('/api/ci/template', data, { params });
	},

	update(id: string, data: PipelineTemplateUpdateReq): Promise<PipelineTemplate> {
		return request.put(`/api/ci/template/${id}`, data);
	},

	resolveVariables(
		data: {
			orchestration: StageOrchestration[]
			variable_declarations?: VariableDeclaration[]
		},
		params: { projectId: string }
	): Promise<VariableDeclaration[]> {
		return request.post('/api/ci/template/resolve-variables', data, { params });
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/ci/template/${id}`);
	},

	duplicate(id: string, params: { projectId: string }): Promise<PipelineTemplate> {
		return request.post(`/api/ci/template/${id}/duplicate`, {}, { params });
	},

	getSnapshot(snapshotId: string, params: { projectId: string }): Promise<PipelineSnapshot> {
		return request.get(`/api/ci/snapshot/${snapshotId}`, { params });
	},
};
