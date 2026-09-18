import type {
  CredentialCreateReq,
  CredentialDetailResp,
  CredentialExportResp,
  CredentialImportReq,
  CredentialPaginatedResp,
  CredentialResp,
  CredentialUpdateReq,
} from '@/gen/proto/orbit/v1/credential/credential';
import request from '@/utils/request';

// Credential API
export const credentialApi = {
  list(
    projectId: string,
    params?: {
      page?: number;
      per_page?: number;
      search?: string;
    }
  ): Promise<CredentialPaginatedResp> {
    return request.get('/api/repository-credential', {
      params: { project_id: projectId, ...params },
    });
  },

  get(projectId: string, id: string): Promise<CredentialDetailResp> {
    return request.get(`/api/repository-credential/${id}`, { params: { project_id: projectId } });
  },

  create(projectId: string, data: CredentialCreateReq): Promise<CredentialResp> {
    return request.post('/api/repository-credential', data, { params: { project_id: projectId } });
  },

  update(projectId: string, id: string, data: CredentialUpdateReq): Promise<CredentialResp> {
    return request.put(`/api/repository-credential/${id}`, data, {
      params: { project_id: projectId },
    });
  },

  delete(projectId: string, id: string): Promise<void> {
    return request.delete(`/api/repository-credential/${id}`, {
      params: { project_id: projectId },
    });
  },

  exportCredential(projectId: string, id: string): Promise<CredentialExportResp> {
    return request.get(`/api/repository-credential/${id}/export`, {
      params: { project_id: projectId },
    });
  },

  importCredential(projectId: string, data: CredentialImportReq): Promise<CredentialResp> {
    return request.post('/api/repository-credential/import', data, {
      params: { project_id: projectId },
    });
  },
};
