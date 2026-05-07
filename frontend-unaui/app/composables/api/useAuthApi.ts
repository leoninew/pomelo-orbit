import type { LoginReq, TokenResp, UserInfo } from '~/types/auth'

export const useAuthApi = () => {
  const api = useApi()

  return {
    login(data: LoginReq) {
      return api.post<TokenResp>('/api/auth/login', data)
    },

    logout() {
      return api.post<Record<string, never>>('/api/auth/logout')
    },

    getCurrentUser() {
      return api.get<UserInfo>('/api/auth/me')
    }
  }
}
