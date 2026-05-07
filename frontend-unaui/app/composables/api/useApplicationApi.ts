import type { PaginatedResp } from '~/types/common'
import type { Application } from '~/types/cd/application'

export const useApplicationApi = () => {
  const api = useApi()

  return {
    list(params?: { page?: number; per_page?: number; search?: string }) {
      return api.get<PaginatedResp<Application>>('/api/cd/applications', params)
    },

    get(id: string) {
      return api.get<Application>(`/api/cd/applications/${id}`)
    },

    deploy(id: string, branch?: string, env?: string) {
      return api.post<{ deployment_id: string }>(`/api/cd/applications/${id}/deploy`, {
        branch,
        env
      })
    },

    stop(id: string, removeVolumes?: boolean) {
      return api.post<{ deployment_id: string }>(`/api/cd/applications/${id}/stop`, {
        remove_volumes: removeVolumes
      })
    },

    restart(id: string) {
      return api.post<{ deployment_id: string }>(`/api/cd/applications/${id}/restart`)
    }
  }
}
