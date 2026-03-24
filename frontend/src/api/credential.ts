import type {
	Credential,
	CredentialCreateReq,
	CredentialUpdateReq,
	PaginatedResp,
} from '@/types/api';
import request from '@/utils/request';

// 凭据相关 API
export const credentialApi = {
	// 获取凭据列表
	list(params?: {
		page?: number
		per_page?: number
		search?: string
	}): Promise<PaginatedResp<Credential>> {
		return request.get('/api/credential', { params });
	},

	// 获取凭据详情
	get(id: string): Promise<Credential> {
		return request.get(`/api/credential/${id}`);
	},

	// 创建凭据
	create(data: CredentialCreateReq): Promise<Credential> {
		return request.post('/api/credential', data);
	},

	// 更新凭据
	update(id: string, data: CredentialUpdateReq): Promise<Credential> {
		return request.put(`/api/credential/${id}`, data);
	},

	// 删除凭据
	delete(id: string): Promise<void> {
		return request.delete(`/api/credential/${id}`);
	},
};
