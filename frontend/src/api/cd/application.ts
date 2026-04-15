import type {
	Application,
	ApplicationCreateReq,
	ApplicationExportResp,
	ApplicationImportReq,
	ApplicationRoute,
	ApplicationUpdateReq,
	ConfigFile,
} from '@/types/cd/application';
import type { PaginatedResp } from '@/types/common';
import request from '@/utils/request';

// 应用相关 API
export const applicationApi = {
	// 获取应用列表
	list(params?: {
		page?: number
		per_page?: number
		search?: string
	}): Promise<PaginatedResp<Application>> {
		return request.get('/api/cd/applications', { params });
	},

	// 获取应用详情
	get(id: string): Promise<Application> {
		return request.get(`/api/cd/applications/${id}`);
	},

	// 创建应用
	create(data: ApplicationCreateReq): Promise<Application> {
		return request.post('/api/cd/applications', data);
	},

	// 更新应用
	update(id: string, data: ApplicationUpdateReq): Promise<Application> {
		return request.put(`/api/cd/applications/${id}`, data);
	},

	// 删除应用
	delete(id: string, removeDir: boolean = false): Promise<void> {
		return request.delete(`/api/cd/applications/${id}`, {
			data: { remove_dir: removeDir },
		});
	},

	// 手动触发部署
	deploy(id: string, branch?: string, env?: string): Promise<{ deployment_id: string }> {
		return request.post(`/api/cd/applications/${id}/deploy`, { branch, env });
	},

	// 停止应用
	stop(id: string, removeVolumes?: boolean): Promise<{ deployment_id: string }> {
		return request.post(`/api/cd/applications/${id}/stop`, {
			remove_volumes: removeVolumes,
		});
	},

	// 重启应用
	restart(id: string): Promise<{ deployment_id: string }> {
		return request.post(`/api/cd/applications/${id}/restart`);
	},

	// 获取应用状态
	getStatus(id: string): Promise<{ status: string; error?: string }> {
		return request.get(`/api/cd/applications/${id}/status`);
	},

	// 获取应用日志
	getLogs(id: string, tail?: number): Promise<{ logs: string; error?: string }> {
		return request.get(`/api/cd/applications/${id}/logs`, { params: { tail } });
	},

	// 读取应用文件
	readFile(id: string, fileId: string): Promise<{ content: string; path: string }> {
		return request.get(`/api/cd/applications/${id}/file/${fileId}`);
	},

	// 写入应用文件
	writeFile(id: string, fileId: string, path: string, content: string): Promise<ConfigFile> {
		return request.put(`/api/cd/applications/${id}/file/${fileId}`, {
			path,
			content,
		});
	},

	// 创建应用文件
	createFile(id: string, path: string, content: string = ''): Promise<ConfigFile> {
		return request.post(`/api/cd/applications/${id}/file`, { path, content });
	},

	// 获取应用文件列表
	listFiles(id: string): Promise<ConfigFile[]> {
		return request.get(`/api/cd/applications/${id}/files`);
	},

	// 删除应用文件
	deleteFile(id: string, fileId: string): Promise<void> {
		return request.delete(`/api/cd/applications/${id}/file/${fileId}`);
	},

	// 导出应用
	exportApplication(id: string): Promise<ApplicationExportResp> {
		return request.get(`/api/cd/applications/${id}/export`);
	},

	// 导入应用
	importApplication(data: ApplicationImportReq): Promise<Application> {
		return request.post('/api/cd/applications/import', data);
	},

	// 获取路由托管列表
	listRoutes(id: string): Promise<ApplicationRoute[]> {
		return request.get(`/api/cd/applications/${id}/route`);
	},

	// 创建路由托管
	createRoute(
		id: string,
		data: { service_name: string; domain: string; port: number }
	): Promise<ApplicationRoute> {
		return request.post(`/api/cd/applications/${id}/route`, data);
	},

	// 更新路由托管
	updateRoute(
		id: string,
		routeId: string,
		data: { service_name: string; domain: string; port: number }
	): Promise<ApplicationRoute> {
		return request.put(`/api/cd/applications/${id}/route/${routeId}`, data);
	},

	// 删除路由托管
	deleteRoute(id: string, routeId: string): Promise<void> {
		return request.delete(`/api/cd/applications/${id}/route/${routeId}`);
	},

	// 解析 docker-compose service 列表
	listComposeServices(id: string): Promise<string[]> {
		return request.get(`/api/cd/applications/${id}/compose-service`);
	},
};
