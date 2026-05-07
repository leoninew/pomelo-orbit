import type { PaginatedResp } from '~/types/common'
import type { PipelineRun } from '~/types/ci/run'

export const usePipelineRunApi = () => {
  const api = useApi()

  return {
    list(params?: {
      page?: number
      per_page?: number
      repository_id?: string
      template_id?: string
    }) {
      return api.get<PaginatedResp<PipelineRun>>('/api/ci/run', params)
    },

    get(id: string) {
      return api.get<PipelineRun>(`/api/ci/run/${id}`)
    },

    retry(id: string) {
      return api.post<PipelineRun>(`/api/ci/run/${id}/retry`)
    },

    cancel(id: string) {
      return api.post<PipelineRun>(`/api/ci/run/${id}/cancel`)
    }
  }
}
