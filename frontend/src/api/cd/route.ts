import type { PaginatedResp } from '@/types/common';
import request from '@/utils/request';

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
	list(page = 1, per_page = 20): Promise<PaginatedResp<Route>> {
		return request.get('/api/cd/routes', { params: { page, per_page } });
	},

	get(id: string): Promise<Route> {
		return request.get(`/api/cd/routes/${id}`);
	},

	create(data: RouteCreateReq): Promise<Route> {
		return request.post('/api/cd/routes', data);
	},

	update(id: string, data: RouteUpdateReq): Promise<Route> {
		return request.put(`/api/cd/routes/${id}`, data);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/cd/routes/${id}`);
	},

	enable(id: string): Promise<void> {
		return request.put(`/api/cd/routes/${id}/enable`);
	},

	disable(id: string): Promise<void> {
		return request.put(`/api/cd/routes/${id}/disable`);
	},

	sync(): Promise<void> {
		return request.post('/api/cd/routes/sync');
	},

	uploadCert(id: string, certFile: File): Promise<Route> {
		const formData = new FormData();
		formData.append('pem', certFile);
		return request.post(`/api/cd/routes/${id}/cert`, formData, {
			headers: { 'Content-Type': 'multipart/form-data' },
		});
	},

	disableHttps(id: string): Promise<Route> {
		return request.delete(`/api/cd/routes/${id}/https`);
	},

	enableLetsencrypt(id: string): Promise<Route> {
		return request.post(`/api/cd/routes/${id}/letsencrypt`);
	},

	enableMkcert(id: string): Promise<Route> {
		return request.post(`/api/cd/routes/${id}/mkcert`);
	},
};
