import type { Job, JobLog } from '@/types/api';
import request from '@/utils/request';

// Job API
export const jobApi = {
	get(id: string): Promise<Job> {
		return request.get(`/api/ci/jobs/${id}`);
	},

	listLogs(jobId: string): Promise<JobLog | null> {
		return request.get(`/api/ci/jobs/${jobId}/logs`);
	},
};
