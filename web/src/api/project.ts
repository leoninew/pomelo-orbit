import type {
  ProjectDeprecateReq,
  ProjectListResp,
  ProjectMemberListResp,
  ProjectMemberReq,
  ProjectResp,
  ProjectSaveReq,
} from '@/gen/orbit/api/v1/project';
import request from '@/utils/request';

export const projectApi = {
  list(): Promise<ProjectListResp> {
    return request.get('/api/project');
  },

  get(id: string): Promise<ProjectResp> {
    return request.get(`/api/project/${id}`);
  },

  create(data: ProjectSaveReq): Promise<ProjectResp> {
    return request.post('/api/project', data);
  },

  update(id: string, data: ProjectSaveReq): Promise<ProjectResp> {
    return request.put(`/api/project/${id}`, data);
  },

  deprecate(id: string, data: ProjectDeprecateReq): Promise<void> {
    return request.post(`/api/project/${id}/deprecate`, data);
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
