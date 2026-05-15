import type { PaginatedResp } from '@/types/common';
import type { Artifact } from '@/types/ci/stage_run';
import request from '@/utils/request';

export const artifactApi = {
	list(params?: {
		page?: number
		per_page?: number
		repository_id?: string
		template_id?: string
		search?: string
		projectId?: string
	}): Promise<PaginatedResp<Artifact>> {
		return request.get('/api/ci/artifact', { params });
	},
};
