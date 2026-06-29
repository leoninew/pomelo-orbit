import type {
  Project,
  ProjectCreateReq,
  ProjectMember,
  ProjectMemberReq,
  ProjectUpdateReq,
} from '@/types/project';
import request from '@/utils/request';

export const projectApi = {
  list(): Promise<Project[]> {
    return request.get('/api/project');
  },

  get(id: string): Promise<Project> {
    return request.get(`/api/project/${id}`);
  },

  create(data: ProjectCreateReq): Promise<Project> {
    return request.post('/api/project', data);
  },

  update(id: string, data: ProjectUpdateReq): Promise<Project> {
    return request.put(`/api/project/${id}`, data);
  },

  deprecate(id: string): Promise<void> {
    return request.post(`/api/project/${id}/deprecate`);
  },

  listMembers(id: string): Promise<ProjectMember[]> {
    return request.get(`/api/project/${id}/member`);
  },

  addMember(id: string, data: ProjectMemberReq): Promise<ProjectMember[]> {
    return request.post(`/api/project/${id}/member`, data);
  },

  removeMember(id: string, userId: string): Promise<ProjectMember[]> {
    return request.delete(`/api/project/${id}/member/${userId}`);
  },
};
