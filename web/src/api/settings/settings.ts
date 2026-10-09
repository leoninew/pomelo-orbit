import type {
  SystemConfigResetReq,
  SystemConfigResp,
  SystemConfigUpdateReq,
} from '@/gen/proto/orbit/v1/settings/settings';
import request from '@/utils/request';

export const settingApi = {
  getConfig(): Promise<SystemConfigResp> {
    return request.get('/api/setting/config');
  },
  updateConfig(data: SystemConfigUpdateReq): Promise<SystemConfigResp> {
    return request.put('/api/setting/config', data);
  },
  resetConfig(data: SystemConfigResetReq): Promise<SystemConfigResp> {
    return request.delete('/api/setting/config', { data });
  },
};
