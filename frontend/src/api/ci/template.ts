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
	list(params?: { page?: number; per_page?: number }): Promise<PaginatedResp<PipelineTemplate>> {
		return request.get('/api/ci/template', { params });
	},

	get(id: string): Promise<PipelineTemplate> {
		return request.get(`/api/ci/template/${id}`);
	},

	create(data: PipelineTemplateCreateReq): Promise<PipelineTemplate> {
		return request.post('/api/ci/template', data);
	},

	update(id: string, data: PipelineTemplateUpdateReq): Promise<PipelineTemplate> {
		return request.put(`/api/ci/template/${id}`, data);
	},

	resolveVariables(data: {
		orchestration: StageOrchestration[]
		variable_declarations?: VariableDeclaration[]
	}): Promise<VariableDeclaration[]> {
		return request.post('/api/ci/template/resolve-variables', data);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/ci/template/${id}`);
	},

	duplicate(id: string): Promise<PipelineTemplate> {
		return request.post(`/api/ci/template/${id}/duplicate`);
	},

	getSnapshot(snapshotId: string): Promise<PipelineSnapshot> {
		return request.get(`/api/ci/snapshot/${snapshotId}`);
	},
};
