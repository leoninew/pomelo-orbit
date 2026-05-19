import type { PaginatedResp } from '@/types/common';
import type { UserCreateReq, UserListResp, UserResp, UserRoleUpdateReq, UserUpdateReq } from '@/types/user';
import request from '@/utils/request';

export const userApi = {
	list(params: {
		page: number
		per_page: number
		search?: string
	}): Promise<PaginatedResp<UserListResp>> {
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

	updateRoles(id: string, data: UserRoleUpdateReq): Promise<UserResp> {
		return request.put(`/api/user/${id}/role`, data);
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
