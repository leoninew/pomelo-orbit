import type {
	Credential,
	CredentialCreateReq,
	CredentialUpdateReq,
	PaginatedResp,
} from '@/types/api';
import request from '@/utils/request';

// Credential API
export const credentialApi = {
	list(params?: { page?: number; per_page?: number }): Promise<PaginatedResp<Credential>> {
		return request.get('/api/v1/ci/credentials', { params });
	},

	get(id: string): Promise<Credential> {
		return request.get(`/api/v1/ci/credentials/${id}`);
	},

	create(data: CredentialCreateReq): Promise<Credential> {
		return request.post('/api/v1/ci/credentials', data);
	},

	update(id: string, data: CredentialUpdateReq): Promise<Credential> {
		return request.put(`/api/v1/ci/credentials/${id}`, data);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/v1/ci/credentials/${id}`);
	},
};
