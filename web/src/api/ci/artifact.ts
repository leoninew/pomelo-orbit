import type { ArtifactPaginatedResp } from '@/gen/proto/orbit/api/v1/artifact';
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
    return request.get('/api/ci/artifact', { params });
  },
};
