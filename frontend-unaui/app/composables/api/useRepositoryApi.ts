import type { PaginatedResp } from '~/types/common'
import type {
  Repository,
  RepositoryListItem,
  RepositoryCreateReq,
  RepositoryUpdateReq
} from '~/types/ci/repository'

export const useRepositoryApi = () => {
  const api = useApi()

  return {
    list(params?: { page?: number; per_page?: number; search?: string }) {
      return api.get<PaginatedResp<RepositoryListItem>>('/api/ci/repository', params)
    },

    get(id: string) {
      return api.get<Repository>(`/api/ci/repository/${id}`)
    },

    create(data: RepositoryCreateReq) {
      return api.post<Repository>('/api/ci/repository', data)
    },

    update(id: string, data: RepositoryUpdateReq) {
      return api.put<Repository>(`/api/ci/repository/${id}`, data)
    },

    delete(id: string) {
      return api.delete<Record<string, never>>(`/api/ci/repository/${id}`)
    },

    trigger(id: string, data?: Record<string, unknown>) {
      return api.post(`/api/ci/repository/${id}/trigger`, data || {})
    }
  }
}
