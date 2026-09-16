import type {
  ArtifactPaginatedResp,
  ArtifactResp,
} from '@/gen/proto/orbit/v1/pipeline_run/artifact';
import request from '@/utils/request';

export const artifactApi = {
  list(
    projectId: string,
    params?: {
      page?: number;
      per_page?: number;
      repository_id?: string;
      pipeline_id?: string;
      search?: string;
    }
  ): Promise<ArtifactPaginatedResp> {
    return request.get('/api/pipeline-run/artifact', {
      params: { project_id: projectId, ...params },
    });
  },

  get(projectId: string, id: string): Promise<ArtifactResp> {
    return request.get(`/api/pipeline-run/artifact/${id}`, { params: { project_id: projectId } });
  },
};
