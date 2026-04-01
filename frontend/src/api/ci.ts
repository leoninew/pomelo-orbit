import type {
	Artifact,
	Credential,
	CredentialCreateReq,
	CredentialUpdateReq,
	Job,
	JobLog,
	PaginatedResp,
	PipelineRun,
	PipelineRunTriggerReq,
	PipelineTemplate,
	PipelineTemplateCreateReq,
	PipelineTemplateUpdateReq,
	Project,
	ProjectCreateReq,
	ProjectUpdateReq,
} from '@/types/api';
import request from '@/utils/request';

// Credential API (导出为 ciCredentialApi 避免与 CD 的 credentialApi 冲突)
export const ciCredentialApi = {
	list(params?: { page?: number; per_page?: number }): Promise<PaginatedResp<Credential>> {
		return request.get('/api/v1/ci/credentials', { params });
	},

	get(id: string): Promise<Credential> {
		return request.get(`/api/v1/ci/credentials/${id}`);
	},

	create(data: CredentialCreateReq): Promise<Credential> {
		return request.post('/api/v1/ci/credentials', data);
	},

	update(id: string, data: CredentialUpdateReq): Promise<Credential> {
		return request.put(`/api/v1/ci/credentials/${id}`, data);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/v1/ci/credentials/${id}`);
	},
};

// PipelineTemplate API
export const pipelineTemplateApi = {
	list(params?: { page?: number; per_page?: number }): Promise<PaginatedResp<PipelineTemplate>> {
		return request.get('/api/v1/ci/templates', { params });
	},

	get(id: string): Promise<PipelineTemplate> {
		return request.get(`/api/v1/ci/templates/${id}`);
	},

	create(data: PipelineTemplateCreateReq): Promise<PipelineTemplate> {
		return request.post('/api/v1/ci/templates', data);
	},

	update(id: string, data: PipelineTemplateUpdateReq): Promise<PipelineTemplate> {
		return request.put(`/api/v1/ci/templates/${id}`, data);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/v1/ci/templates/${id}`);
	},
};

// Project API
export const projectApi = {
	list(params?: { page?: number; per_page?: number }): Promise<PaginatedResp<Project>> {
		return request.get('/api/v1/ci/projects', { params });
	},

	get(id: string): Promise<Project> {
		return request.get(`/api/v1/ci/projects/${id}`);
	},

	create(data: ProjectCreateReq): Promise<Project> {
		return request.post('/api/v1/ci/projects', data);
	},

	update(id: string, data: ProjectUpdateReq): Promise<Project> {
		return request.put(`/api/v1/ci/projects/${id}`, data);
	},

	delete(id: string): Promise<void> {
		return request.delete(`/api/v1/ci/projects/${id}`);
	},

	trigger(id: string, data?: PipelineRunTriggerReq): Promise<PipelineRun> {
		return request.post(`/api/v1/ci/projects/${id}/trigger`, data || {});
	},
};

// PipelineRun API
export const pipelineRunApi = {
	list(params?: {
		page?: number
		per_page?: number
		project_id?: string
	}): Promise<PaginatedResp<PipelineRun>> {
		return request.get('/api/v1/ci/runs', { params });
	},

	get(id: string): Promise<PipelineRun> {
		return request.get(`/api/v1/ci/runs/${id}`);
	},

	retry(id: string): Promise<PipelineRun> {
		return request.post(`/api/v1/ci/runs/${id}/retry`);
	},

	cancel(id: string): Promise<PipelineRun> {
		return request.post(`/api/v1/ci/runs/${id}/cancel`);
	},

	listJobs(runId: string): Promise<Job[]> {
		return request.get(`/api/v1/ci/runs/${runId}/jobs`);
	},

	listArtifacts(runId: string): Promise<Artifact[]> {
		return request.get(`/api/v1/ci/runs/${runId}/artifacts`);
	},
};

// Job API
export const jobApi = {
	get(id: string): Promise<Job> {
		return request.get(`/api/v1/ci/jobs/${id}`);
	},

	listLogs(jobId: string): Promise<JobLog | null> {
		return request.get(`/api/v1/ci/jobs/${jobId}/logs`);
	},
};
