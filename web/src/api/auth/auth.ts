import type {
  CSRFTokenResp,
  GoogleCallbackReq,
  MCPAccessTokenCreatedResp,
  MCPAccessTokenCreateReq,
  MCPAccessTokenListResp,
  LoginHistoryPaginatedResp,
  LoginReq,
  LogoutReq,
  PasswordChangeReq,
  TokenResp,
  TurnstileConfigResp,
  UserInfoResp,
} from '@/gen/proto/orbit/v1/auth/auth';
import request from '@/utils/request';

// 认证相关 API
export const authApi = {
  // 获取 CSRF Token
  getCsrfToken(): Promise<CSRFTokenResp> {
    return request.get('/api/auth/csrf-token');
  },

  // 获取 Turnstile 配置
  getTurnstileConfig(): Promise<TurnstileConfigResp> {
    return request.get('/api/auth/turnstile-config');
  },

  // 登录
  login(data: LoginReq): Promise<TokenResp> {
    return request.post('/api/auth/login', data);
  },

  // 登出
  logout(data: LogoutReq): Promise<void> {
    return request.post('/api/auth/logout', data);
  },

  // 获取当前用户信息
  getCurrentUser(): Promise<UserInfoResp> {
    return request.get('/api/auth/me');
  },

  // 修改密码
  changePassword(data: PasswordChangeReq): Promise<void> {
    return request.put('/api/auth/password', data);
  },

  // 获取登录历史
  listLoginHistory(params?: {
    page?: number;
    per_page?: number;
    search?: string;
  }): Promise<LoginHistoryPaginatedResp> {
    return request.get('/api/auth/login-history', { params });
  },

  listMcpAccessTokens(): Promise<MCPAccessTokenListResp> {
    return request.get('/api/auth/mcp-access-token');
  },

  createMcpAccessToken(data: MCPAccessTokenCreateReq): Promise<MCPAccessTokenCreatedResp> {
    return request.post('/api/auth/mcp-access-token', data);
  },

  revokeMcpAccessToken(id: string): Promise<void> {
    return request.delete(`/api/auth/mcp-access-token/${id}`);
  },

  // Google OAuth 回调
  googleCallback(data: GoogleCallbackReq): Promise<TokenResp> {
    return request.post('/api/auth/google/callback', data);
  },
};
