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
    return request.get('/api/credential', { params });
  },

  get(id: string): Promise<CredentialDetailResp> {
    return request.get(`/api/credential/${id}`);
  },

  create(data: CredentialCreateReq, params: { project_id: string }): Promise<CredentialResp> {
    return request.post('/api/credential', data, { params });
  },

  update(id: string, data: CredentialUpdateReq): Promise<CredentialResp> {
    return request.put(`/api/credential/${id}`, data);
  },

  delete(id: string): Promise<void> {
    return request.delete(`/api/credential/${id}`);
  },

  exportCredential(id: string): Promise<CredentialExportResp> {
    return request.get(`/api/credential/${id}/export`);
  },

  importCredential(
    data: CredentialImportReq,
    params: { project_id: string }
  ): Promise<CredentialResp> {
    return request.post('/api/credential/import', data, { params });
  },
};
