import type {
  ProjectCreateReq,
  ProjectDeprecateReq,
  ProjectListResp,
  ProjectMemberListResp,
  ProjectMemberReq,
  ProjectResp,
  ProjectSaveReq,
} from '@/gen/proto/orbit/v1/project/project';
import request from '@/utils/request';

export const projectApi = {
  list(): Promise<ProjectListResp> {
    return request.get('/api/project');
  },

  get(id: string): Promise<ProjectResp> {
    return request.get(`/api/project/${id}`);
  },

  create(data: ProjectCreateReq): Promise<ProjectResp> {
    return request.post('/api/project', data);
  },

  update(id: string, data: ProjectSaveReq): Promise<ProjectResp> {
    return request.put(`/api/project/${id}`, data);
  },

  deprecate(id: string, data: ProjectDeprecateReq): Promise<void> {
    return request.post(`/api/project/${id}/deprecate`, data);
  },

  exportHandover(id: string): Promise<Blob> {
    return request.get(`/api/project/${id}/handover`, { responseType: 'blob' });
  },

  importHandover(
    file: File,
    input: {
      mode: 'new' | 'replace';
      name?: string;
      code?: string;
      targetProjectId?: string;
      decryptionKey?: string;
      overrideEnvironment: boolean;
    }
  ): Promise<ProjectResp> {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('mode', input.mode);
    if (input.mode === 'new') {
      formData.append('name', input.name ?? '');
      formData.append('code', input.code ?? '');
    }
    formData.append('decryption_key', input.decryptionKey ?? '');
    formData.append('override_environment', String(input.overrideEnvironment));
    const path =
      input.mode === 'replace'
        ? `/api/project/${input.targetProjectId ?? ''}/handover`
        : '/api/project/handover';
    return request.post(path, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },

  listMembers(id: string): Promise<ProjectMemberListResp> {
    return request.get(`/api/project/${id}/member`);
  },

  addMember(id: string, data: ProjectMemberReq): Promise<ProjectMemberListResp> {
    return request.post(`/api/project/${id}/member`, data);
  },

  removeMember(id: string, userId: string): Promise<ProjectMemberListResp> {
    return request.delete(`/api/project/${id}/member/${userId}`);
  },
};
