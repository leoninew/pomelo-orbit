import type { PaginatedResp } from '~/types/common'
import type { Deployment } from '~/types/cd/deployment'

export const useDeploymentApi = () => {
  const api = useApi()

  return {
    list(params?: {
      page?: number
      per_page?: number
      application_id?: string
      status?: string
      search?: string
      date_from?: string
      date_to?: string
    }) {
      return api.get<PaginatedResp<Deployment>>('/api/cd/deployments', params)
    },

    get(id: string) {
      return api.get<Deployment>(`/api/cd/deployments/${id}`)
    },

    cancel(id: string) {
      return api.post<Record<string, never>>(`/api/cd/deployments/${id}/cancel`)
    }
  }
}
