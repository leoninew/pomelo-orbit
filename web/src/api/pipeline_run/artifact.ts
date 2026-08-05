import type {
  ArtifactPaginatedResp,
  ArtifactResp,
} from '@/gen/proto/orbit/v1/pipeline_run/artifact';
import request from '@/utils/request';

export const artifactApi = {
  list(params?: {
    page?: number;
    per_page?: number;
    repository_id?: string;
    template_id?: string;
    search?: string;
    project_id?: string;
  }): Promise<ArtifactPaginatedResp> {
    return request.get('/api/pipeline-run/artifact', { params });
  },

  get(id: string): Promise<ArtifactResp> {
    return request.get(`/api/pipeline-run/artifact/${id}`);
  },
};
