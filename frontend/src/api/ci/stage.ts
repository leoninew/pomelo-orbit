import type { StageLog } from '@/types/ci';
import request from '@/utils/request';

export const stageApi = {
	getLog(stageRunId: string): Promise<StageLog | null> {
		return request.get(`/api/ci/stages/${stageRunId}/log`);
	},
};
