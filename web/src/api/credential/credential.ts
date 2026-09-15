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
  list(params?: {
    page?: number;
    per_page?: number;
    search?: string;
    project_id?: string;
  }): Promise<CredentialPaginatedResp> {
    return request.get('/api/repository-credential', { params });
  },

  get(id: string): Promise<CredentialDetailResp> {
    return request.get(`/api/repository-credential/${id}`);
  },

  create(data: CredentialCreateReq, params: { project_id: string }): Promise<CredentialResp> {
    return request.post('/api/repository-credential', data, { params });
  },

  update(id: string, data: CredentialUpdateReq): Promise<CredentialResp> {
    return request.put(`/api/repository-credential/${id}`, data);
  },

  delete(id: string): Promise<void> {
    return request.delete(`/api/repository-credential/${id}`);
  },

  exportCredential(id: string): Promise<CredentialExportResp> {
    return request.get(`/api/repository-credential/${id}/export`);
  },

  importCredential(
    data: CredentialImportReq,
    params: { project_id: string }
  ): Promise<CredentialResp> {
    return request.post('/api/repository-credential/import', data, { params });
  },
};
