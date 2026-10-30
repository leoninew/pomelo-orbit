import type {
  RecordCreateReq,
  RecordPaginatedResp,
  RecordResp,
  RecordUpdateReq,
  ZoneListResp,
} from '@/gen/proto/orbit/v1/dns/dns';
import request from '@/utils/request';

export const dnsApi = {
  zones(projectId: string): Promise<ZoneListResp> {
    return request.get('/api/dns/zone', { params: { project_id: projectId } });
  },
  records(
    projectId: string,
    zoneId: string,
    page: number,
    perPage: number,
    search: string
  ): Promise<RecordPaginatedResp> {
    return request.get(`/api/dns/zone/${zoneId}/record`, {
      params: { project_id: projectId, page, per_page: perPage, search: search || undefined },
    });
  },
  create(projectId: string, zoneId: string, data: RecordCreateReq): Promise<RecordResp> {
    return request.post(`/api/dns/zone/${zoneId}/record`, data, {
      params: { project_id: projectId },
    });
  },
  delete(projectId: string, zoneId: string, recordId: string): Promise<void> {
    return request.delete(`/api/dns/zone/${zoneId}/record/${recordId}`, {
      params: { project_id: projectId },
    });
  },
  update(
    projectId: string,
    zoneId: string,
    recordId: string,
    data: RecordUpdateReq
  ): Promise<RecordResp> {
    return request.put(`/api/dns/zone/${zoneId}/record/${recordId}`, data, {
      params: { project_id: projectId },
    });
  },
};
