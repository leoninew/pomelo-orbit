import type { PaginatedResp } from '@/types/common';
import type { PermissionResp, RoleCreateReq, RoleResp, RoleUpdateReq } from '@/types/role';
import request from '@/utils/request';

export const roleApi = {
  list(params: {
    page: number;
    per_page: number;
    search?: string;
  }): Promise<PaginatedResp<RoleResp>> {
    return request.get('/api/role', { params });
  },
  get(id: string): Promise<RoleResp> {
    return request.get(`/api/role/${id}`);
  },
  listPermissions(): Promise<PermissionResp[]> {
    return request.get('/api/role/permission');
  },
  create(data: RoleCreateReq): Promise<RoleResp> {
    return request.post('/api/role', data);
  },
  update(id: string, data: RoleUpdateReq): Promise<RoleResp> {
    return request.put(`/api/role/${id}`, data);
  },
  delete(id: string): Promise<void> {
    return request.delete(`/api/role/${id}`);
  },
};
