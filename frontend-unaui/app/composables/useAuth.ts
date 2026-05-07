import type { UserInfo } from '~/types/auth'
import { useAuthApi } from '~/composables/api'

export const useAuth = () => {
  const token = useState<string | null>('auth-token', () => null)
  const user = useState<UserInfo | null>('auth-user', () => null)

  const isAuthenticated = computed(() => !!token.value)

  const authApi = useAuthApi()

  async function login(username: string, password: string) {
    const response = await authApi.login({ username, password })
    token.value = response.access_token

    // 保存到 localStorage
    if (typeof window !== 'undefined') {
      localStorage.setItem('token', response.access_token)
    }

    // 获取用户信息
    await fetchUser()
  }

  async function logout() {
    try {
      await authApi.logout()
    } catch (error) {
      console.error('Logout error:', error)
    } finally {
      token.value = null
      user.value = null

      if (typeof window !== 'undefined') {
        localStorage.removeItem('token')
        localStorage.removeItem('user')
      }

      navigateTo('/login')
    }
  }

  async function fetchUser() {
    try {
      user.value = await authApi.getCurrentUser()

      if (typeof window !== 'undefined') {
        localStorage.setItem('user', JSON.stringify(user.value))
      }
    } catch (error) {
      console.error('Fetch user error:', error)
      token.value = null
      user.value = null
    }
  }

  function initAuth() {
    if (typeof window !== 'undefined') {
      const savedToken = localStorage.getItem('token')
      const savedUser = localStorage.getItem('user')

      if (savedToken) {
        token.value = savedToken
      }

      if (savedUser) {
        try {
          user.value = JSON.parse(savedUser)
        } catch (error) {
          console.error('Parse user error:', error)
        }
      }
    }
  }

  return {
    token,
    user,
    isAuthenticated,
    login,
    logout,
    fetchUser,
    initAuth
  }
}
