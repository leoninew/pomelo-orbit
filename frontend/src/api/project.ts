import type { Project, ProjectCreateReq, ProjectUpdateReq } from '@/types/project';
import request from '@/utils/request';

export const projectApi = {
	list(): Promise<Project[]> {
		return request.get('/api/project');
	},

	get(id: string): Promise<Project> {
		return request.get(`/api/project/${id}`);
	},

	create(data: ProjectCreateReq): Promise<Project> {
		return request.post('/api/project', data);
	},

	update(id: string, data: ProjectUpdateReq): Promise<Project> {
		return request.put(`/api/project/${id}`, data);
	},
};
