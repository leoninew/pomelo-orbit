import type {
	Application,
	ApplicationCreateReq,
	ApplicationExportResp,
	ApplicationImportReq,
	ApplicationRoute,
	ApplicationServiceConfig,
	ApplicationUpdateReq,
	ComposeServiceResp,
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
		return request.get('/api/cd/application', { params });
	},

	// 获取应用详情
	get(id: string): Promise<Application> {
		return request.get(`/api/cd/application/${id}`);
	},

	// 创建应用
	create(data: ApplicationCreateReq): Promise<Application> {
		return request.post('/api/cd/application', data);
	},

	// 更新应用
	update(id: string, data: ApplicationUpdateReq): Promise<Application> {
		return request.put(`/api/cd/application/${id}`, data);
	},

	// 删除应用
	delete(id: string, removeDir: boolean = false): Promise<void> {
		return request.delete(`/api/cd/application/${id}`, {
			data: { remove_dir: removeDir },
		});
	},

	// 手动触发部署
	deploy(id: string): Promise<{ deployment_id: string }> {
		return request.post(`/api/cd/application/${id}/deploy`, {});
	},

	// 停止应用
	stop(id: string, removeVolumes?: boolean): Promise<{ deployment_id: string }> {
		return request.post(`/api/cd/application/${id}/stop`, {
			remove_volumes: removeVolumes,
		});
	},

	// 重启应用
	restart(id: string): Promise<{ deployment_id: string }> {
		return request.post(`/api/cd/application/${id}/restart`);
	},

	// 获取应用状态
	getStatus(id: string): Promise<{ status: string; error?: string }> {
		return request.get(`/api/cd/application/${id}/status`);
	},

	// 获取应用日志
	getLogs(id: string, tail?: number): Promise<{ logs: string; error?: string }> {
		return request.get(`/api/cd/application/${id}/logs`, { params: { tail } });
	},

	// 读取应用文件
	readFile(id: string, fileId: string): Promise<{ content: string; path: string }> {
		return request.get(`/api/cd/application/${id}/file/${fileId}`);
	},

	// 写入应用文件
	writeFile(id: string, fileId: string, path: string, content: string): Promise<ConfigFile> {
		return request.put(`/api/cd/application/${id}/file/${fileId}`, {
			path,
			content,
		});
	},

	// 创建应用文件
	createFile(id: string, path: string, content: string = ''): Promise<ConfigFile> {
		return request.post(`/api/cd/application/${id}/file`, { path, content });
	},

	// 获取应用文件列表
	listFiles(id: string): Promise<ConfigFile[]> {
		return request.get(`/api/cd/application/${id}/files`);
	},

	/** 预览部署时生成的 docker-compose.yml（模板渲染、镜像覆盖、路由 labels） */
	previewCompose(id: string): Promise<{ compose_yaml: string }> {
		return request.post(`/api/cd/application/${id}/compose-preview`);
	},

	// 删除应用文件
	deleteFile(id: string, fileId: string): Promise<void> {
		return request.delete(`/api/cd/application/${id}/file/${fileId}`);
	},

	// 导出应用
	exportApplication(id: string): Promise<ApplicationExportResp> {
		return request.get(`/api/cd/application/${id}/export`);
	},

	// 导入应用
	importApplication(data: ApplicationImportReq): Promise<Application> {
		return request.post('/api/cd/application/import', data);
	},

	// 获取路由托管列表
	listRoutes(id: string): Promise<ApplicationRoute[]> {
		return request.get(`/api/cd/application/${id}/route`);
	},

	// 创建路由托管
	createRoute(
		id: string,
		data: { service_name: string; domain: string; port: number }
	): Promise<ApplicationRoute> {
		return request.post(`/api/cd/application/${id}/route`, data);
	},

	// 更新路由托管
	updateRoute(
		id: string,
		routeId: string,
		data: { service_name: string; domain: string; port: number }
	): Promise<ApplicationRoute> {
		return request.put(`/api/cd/application/${id}/route/${routeId}`, data);
	},

	// 删除路由托管
	deleteRoute(id: string, routeId: string): Promise<void> {
		return request.delete(`/api/cd/application/${id}/route/${routeId}`);
	},

	// 解析 docker-compose service 列表
	listComposeServices(id: string): Promise<ComposeServiceResp[]> {
		return request.get(`/api/cd/application/${id}/compose-service`);
	},

	// 获取 service 级配置
	listServiceConfigs(id: string): Promise<ApplicationServiceConfig[]> {
		return request.get(`/api/cd/application/${id}/service-config`);
	},

	// 更新 service 级配置
	updateServiceConfig(
		id: string,
		serviceName: string,
		image: string | null
	): Promise<ApplicationServiceConfig> {
		return request.put(`/api/cd/application/${id}/service-config/${serviceName}`, { image });
	},
};
