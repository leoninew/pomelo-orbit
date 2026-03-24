import type {
	Application,
	ApplicationCreateReq,
	ApplicationUpdateReq,
	ConfigFile,
	Credential,
	PaginatedResp,
} from '@/types/api';
import request from '@/utils/request';

// 应用相关 API
export const applicationApi = {
	// 获取应用列表
	list(params?: {
		page?: number
		per_page?: number
		search?: string
	}): Promise<PaginatedResp<Application>> {
		return request.get('/api/application', { params });
	},

	// 获取应用详情
	get(id: string): Promise<Application> {
		return request.get(`/api/application/${id}`);
	},

	// 创建应用
	create(data: ApplicationCreateReq): Promise<Application> {
		return request.post('/api/application', data);
	},

	// 更新应用
	update(id: string, data: ApplicationUpdateReq): Promise<Application> {
		return request.put(`/api/application/${id}`, data);
	},

	// 删除应用
	delete(id: string, removeDir: boolean = false): Promise<void> {
		return request.delete(`/api/application/${id}`, { data: { remove_dir: removeDir } });
	},

	// 手动触发部署
	deploy(id: string, branch?: string, env?: string): Promise<{ deployment_id: string }> {
		return request.post(`/api/application/${id}/deploy`, { branch, env });
	},

	// 停止应用
	stop(id: string, removeVolumes?: boolean): Promise<{ deployment_id: string }> {
		return request.post(`/api/application/${id}/stop`, { remove_volumes: removeVolumes });
	},

	// 重启应用
	restart(id: string): Promise<{ deployment_id: string }> {
		return request.post(`/api/application/${id}/restart`);
	},

	// 获取应用状态
	getStatus(id: string): Promise<{ status: string; error?: string }> {
		return request.get(`/api/application/${id}/status`);
	},

	// 获取应用日志
	getLogs(id: string, tail?: number): Promise<{ logs: string; error?: string }> {
		return request.get(`/api/application/${id}/logs`, { params: { tail } });
	},

	// 读取应用文件
	readFile(id: string, fileId: string): Promise<{ content: string; path: string }> {
		return request.get(`/api/application/${id}/file/${fileId}`);
	},

	// 写入应用文件
	writeFile(id: string, fileId: string, path: string, content: string): Promise<ConfigFile> {
		return request.put(`/api/application/${id}/file/${fileId}`, { path, content });
	},

	// 创建应用文件
	createFile(id: string, path: string, content: string = ''): Promise<ConfigFile> {
		return request.post(`/api/application/${id}/file`, { path, content });
	},

	// 获取应用文件列表
	listFiles(id: string): Promise<ConfigFile[]> {
		return request.get(`/api/application/${id}/files`);
	},

	// 删除应用文件
	deleteFile(id: string, fileId: string): Promise<void> {
		return request.delete(`/api/application/${id}/file/${fileId}`);
	},

	// 获取应用凭据
	getCredential(id: string): Promise<Credential | null> {
		return request.get(`/api/application/${id}/credential`);
	},
};
