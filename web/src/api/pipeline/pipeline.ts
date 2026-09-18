import type {
  PipelineCreateReq,
  PipelineInstantiateReq,
  PipelinePaginatedResp,
  PipelineResp,
  PipelineUpdateReq,
} from '@/gen/proto/orbit/v1/pipeline/pipeline';
import type {
  PipelineStageImportReq,
  PipelineStageNodeUpdateReq,
  PipelineStageTemplateUpdatePreviewResp,
  PipelineStageTemplateUpdateReq,
} from '@/gen/proto/orbit/v1/pipeline/pipeline_stage';
import type { PipelineSnapshotResp } from '@/gen/proto/orbit/v1/pipeline/snapshot';
import request from '@/utils/request';

export const pipelineApi = {
  list(
    projectId: string,
    params?: {
      page?: number;
      per_page?: number;
      search?: string;
      kind?: string;
    }
  ): Promise<PipelinePaginatedResp> {
    return request.get('/api/pipeline', { params: { project_id: projectId, ...params } });
  },

  get(projectId: string, id: string): Promise<PipelineResp> {
    return request.get(`/api/pipeline/${id}`, { params: { project_id: projectId } });
  },

  create(projectId: string, data: PipelineCreateReq): Promise<PipelineResp> {
    return request.post('/api/pipeline', data, { params: { project_id: projectId } });
  },

  update(projectId: string, id: string, data: PipelineUpdateReq): Promise<PipelineResp> {
    return request.put(`/api/pipeline/${id}`, data, { params: { project_id: projectId } });
  },

  delete(projectId: string, id: string): Promise<void> {
    return request.delete(`/api/pipeline/${id}`, { params: { project_id: projectId } });
  },

  instantiate(projectId: string, id: string, data: PipelineInstantiateReq): Promise<PipelineResp> {
    return request.post(`/api/pipeline/${id}/instantiate`, data, {
      params: { project_id: projectId },
    });
  },

  importStage(projectId: string, id: string, data: PipelineStageImportReq): Promise<PipelineResp> {
    return request.post(`/api/pipeline/${id}/stage`, data, { params: { project_id: projectId } });
  },

  updateStage(
    projectId: string,
    pipelineId: string,
    stageId: string,
    data: PipelineStageNodeUpdateReq
  ): Promise<PipelineResp> {
    return request.put(`/api/pipeline/${pipelineId}/stage/${stageId}`, data, {
      params: { project_id: projectId },
    });
  },

  deleteStage(projectId: string, pipelineId: string, stageId: string): Promise<PipelineResp> {
    return request.delete(`/api/pipeline/${pipelineId}/stage/${stageId}`, {
      params: { project_id: projectId },
    });
  },

  previewStageTemplateUpdate(
    projectId: string,
    pipelineId: string,
    stageId: string
  ): Promise<PipelineStageTemplateUpdatePreviewResp> {
    return request.get(`/api/pipeline/${pipelineId}/stage/${stageId}/template-update-preview`, {
      params: { project_id: projectId },
    });
  },

  updateStageTemplate(
    projectId: string,
    pipelineId: string,
    stageId: string,
    data: PipelineStageTemplateUpdateReq
  ): Promise<PipelineResp> {
    return request.post(`/api/pipeline/${pipelineId}/stage/${stageId}/template-update`, data, {
      params: { project_id: projectId },
    });
  },

  getSnapshot(projectId: string, id: string): Promise<PipelineSnapshotResp> {
    return request.get(`/api/pipeline/snapshot/${id}`, { params: { project_id: projectId } });
  },
};
