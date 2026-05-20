import request from '@/utils/request';

export interface TraefikRouter {
	name: string
	provider: string
	status: string
	rule: string
	service: string
	entrypoints: string[]
	tls: boolean
}

export interface TraefikRouteListResp {
	items: TraefikRouter[]
	total: number
}

export interface TraefikConfig {
	dashboard_domain: string
	https_enabled: boolean
}

export const traefikRouteApi = {
	list(params: { project_id: string }): Promise<TraefikRouteListResp> {
		return request.get('/api/cd/traefik-route', { params });
	},
	getConfig(params: { project_id: string }): Promise<TraefikConfig> {
		return request.get('/api/cd/traefik-route/config', { params });
	},
};
