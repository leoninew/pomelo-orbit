import type {
	GoogleCallbackReq,
	LoginHistory,
	LoginReq,
	PasswordChangeReq,
	TokenResp,
	UserInfo,
} from '@/types/auth';
import type { PaginatedResp } from '@/types/common';
import request from '@/utils/request';

// 认证相关 API
export const authApi = {
	// 登录
	login(data: LoginReq): Promise<TokenResp> {
		return request.post('/api/auth/login', data);
	},

	// 登出
	logout(): Promise<void> {
		return request.post('/api/auth/logout');
	},

	// 获取当前用户信息
	getCurrentUser(): Promise<UserInfo> {
		return request.get('/api/auth/me');
	},

	// 修改密码
	changePassword(data: PasswordChangeReq): Promise<void> {
		return request.put('/api/auth/password', data);
	},

	// 获取登录历史
	listLoginHistory(params?: {
		page?: number
		per_page?: number
		search?: string
	}): Promise<PaginatedResp<LoginHistory>> {
		return request.get('/api/auth/login-history', { params });
	},

	// Google OAuth 回调
	googleCallback(data: GoogleCallbackReq): Promise<TokenResp> {
		return request.post('/api/auth/google/callback', data);
	},
};
