import request from '@/utils/request';
import type { PaginatedResp } from '@/types/api';

export interface Route {
	id: string
	name: string
	domain: string
	path_prefix: string
	target_url: string
	enabled: boolean
	https_enabled: boolean
	cert_type: string // manual / letsencrypt
	created_at: string
	updated_at: string
}

export interface RouteCreateReq {
	name: string
	domain: string
	path_prefix: string
	target_url: string
	enabled: boolean
}

export interface RouteUpdateReq {
	name?: string
	domain?: string
	path_prefix?: string
	target_url?: string
	enabled?: boolean
}

export const routeApi = {
	list(page = 1, per_page = 100): Promise<PaginatedResp<Route>> {
		return request.get('/api/route', { params: { page, per_page } });
	},

	get(id: string): Promise<Route> {
		return request.get(`/api/route/${id}`);
	},

	create(data: RouteCreateReq): Promise<Route> {
		return request.post('/api/route', data);
	},

	update(id: string, data: RouteUpdateReq): Promise<Route> {
		return request.put(`/api/route/${id}`, data);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/route/${id}`);
	},

	enable(id: string): Promise<void> {
		return request.put(`/api/route/${id}/enable`);
	},

	disable(id: string): Promise<void> {
		return request.put(`/api/route/${id}/disable`);
	},

	sync(): Promise<void> {
		return request.post('/api/route/sync');
	},

	uploadCert(id: string, certFile: File): Promise<Route> {
		const formData = new FormData();
		formData.append('pem', certFile);
		return request.post(`/api/route/${id}/cert`, formData, {
			headers: { 'Content-Type': 'multipart/form-data' },
		});
	},

	disableHttps(id: string): Promise<Route> {
		return request.delete(`/api/route/${id}/https`);
	},

	enableLetsencrypt(id: string): Promise<Route> {
		return request.post(`/api/route/${id}/letsencrypt`);
	},

	enableMkcert(id: string): Promise<Route> {
		return request.post(`/api/route/${id}/mkcert`);
	},
};
