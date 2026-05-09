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
	list(params?: {
		page?: number
		per_page?: number
		search?: string
	}): Promise<PaginatedResp<Route>> {
		return request.get('/api/cd/route', { params });
	},

	get(id: string): Promise<Route> {
		return request.get(`/api/cd/route/${id}`);
	},

	create(data: RouteCreateReq): Promise<Route> {
		return request.post('/api/cd/route', data);
	},

	update(id: string, data: RouteUpdateReq): Promise<Route> {
		return request.put(`/api/cd/route/${id}`, data);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/cd/route/${id}`);
	},

	enable(id: string): Promise<void> {
		return request.put(`/api/cd/route/${id}/enable`);
	},

	disable(id: string): Promise<void> {
		return request.put(`/api/cd/route/${id}/disable`);
	},

	sync(): Promise<void> {
		return request.post('/api/cd/route/sync');
	},

	uploadCert(id: string, certFile: File): Promise<Route> {
		const formData = new FormData();
		formData.append('pem', certFile);
		return request.post(`/api/cd/route/${id}/cert`, formData, {
			headers: { 'Content-Type': 'multipart/form-data' },
		});
	},

	disableHttps(id: string): Promise<Route> {
		return request.delete(`/api/cd/route/${id}/https`);
	},

	enableLetsencrypt(id: string): Promise<Route> {
		return request.post(`/api/cd/route/${id}/letsencrypt`);
	},

	enableMkcert(id: string): Promise<Route> {
		return request.post(`/api/cd/route/${id}/mkcert`);
	},
};
