import type { PaginatedResp } from '@/types/common';
import type { UserResp, UserCreateReq, UserUpdateReq } from '@/types/user';
import request from '@/utils/request';

export const userApi = {
	list(params: {
		page: number
		per_page: number
		search?: string
	}): Promise<PaginatedResp<UserResp>> {
		return request.get('/api/user', { params });
	},

	get(id: string): Promise<UserResp> {
		return request.get(`/api/user/${id}`);
	},

	create(data: UserCreateReq): Promise<UserResp> {
		return request.post('/api/user', data);
	},

	update(id: string, data: UserUpdateReq): Promise<UserResp> {
		return request.put(`/api/user/${id}`, data);
	},

	disable(id: string): Promise<void> {
		return request.post(`/api/user/${id}/disable`);
	},

	enable(id: string): Promise<void> {
		return request.post(`/api/user/${id}/enable`);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/user/${id}`);
	},
};
