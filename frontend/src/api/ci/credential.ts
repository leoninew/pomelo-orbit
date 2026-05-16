import type {
	Credential,
	CredentialCreateReq,
	CredentialExportResp,
	CredentialImportReq,
	CredentialUpdateReq,
} from '@/types/ci/credential';
import type { PaginatedResp } from '@/types/common';
import request from '@/utils/request';

// Credential API
export const credentialApi = {
	list(params?: {
		page?: number
		per_page?: number
		search?: string
		project_id?: string
	}): Promise<PaginatedResp<Credential>> {
		return request.get('/api/ci/credential', { params });
	},

	get(id: string): Promise<Credential> {
		return request.get(`/api/ci/credential/${id}`);
	},

	create(data: CredentialCreateReq, params: { project_id: string }): Promise<Credential> {
		return request.post('/api/ci/credential', data, { params });
	},

	update(id: string, data: CredentialUpdateReq): Promise<Credential> {
		return request.put(`/api/ci/credential/${id}`, data);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/ci/credential/${id}`);
	},

	exportCredential(id: string): Promise<CredentialExportResp> {
		return request.get(`/api/ci/credential/${id}/export`);
	},

	importCredential(data: CredentialImportReq, params: { project_id: string }): Promise<Credential> {
		return request.post('/api/ci/credential/import', data, { params });
	},
};
