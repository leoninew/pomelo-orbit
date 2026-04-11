import type {
	SystemConfigResetReq,
	SystemConfigResp,
	SystemConfigUpdateReq,
} from '@/types/cd/settings';
import request from '@/utils/request';

export const settingApi = {
	getConfig(): Promise<SystemConfigResp> {
		return request.get('/api/settings/config');
	},
	updateConfig(data: SystemConfigUpdateReq): Promise<SystemConfigResp> {
		return request.put('/api/settings/config', data);
	},
	resetConfig(data: SystemConfigResetReq): Promise<SystemConfigResp> {
		return request.delete('/api/settings/config', { data });
	},
};
