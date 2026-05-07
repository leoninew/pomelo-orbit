interface FetchOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
  body?: unknown
  params?: Record<string, unknown>
}

export const useApi = () => {
  const config = useRuntimeConfig()

  const apiFetch = async <T>(url: string, options: FetchOptions = {}): Promise<T> => {
    const { method = 'GET', body, params } = options

    // 直接从 localStorage 读取 token，避免循环依赖
    const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null

    return $fetch<T>(url, {
      baseURL: (config.public.apiBaseUrl as string) || '',
      method,
      body: body as BodyInit,
      params,
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {})
      },
      onResponseError({ response }) {
        // 处理错误
        if (response.status === 401) {
          // 清除认证状态
          if (typeof window !== 'undefined') {
            localStorage.removeItem('token')
            localStorage.removeItem('user')
          }
          navigateTo('/login')
        }
      }
    })
  }

  return {
    get: <T>(url: string, params?: Record<string, unknown>) =>
      apiFetch<T>(url, { method: 'GET', params }),

    post: <T>(url: string, body?: unknown) => apiFetch<T>(url, { method: 'POST', body }),

    put: <T>(url: string, body?: unknown) => apiFetch<T>(url, { method: 'PUT', body }),

    delete: <T>(url: string) => apiFetch<T>(url, { method: 'DELETE' })
  }
}
