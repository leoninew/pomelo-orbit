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
  list(params?: {
    page?: number;
    per_page?: number;
    search?: string;
    project_id?: string;
    kind?: string;
  }): Promise<PipelinePaginatedResp> {
    return request.get('/api/pipeline', { params });
  },

  get(id: string): Promise<PipelineResp> {
    return request.get(`/api/pipeline/${id}`);
  },

  create(data: PipelineCreateReq, params: { project_id: string }): Promise<PipelineResp> {
    return request.post('/api/pipeline', data, { params });
  },

  update(id: string, data: PipelineUpdateReq): Promise<PipelineResp> {
    return request.put(`/api/pipeline/${id}`, data);
  },

  delete(id: string): Promise<void> {
    return request.delete(`/api/pipeline/${id}`);
  },

  instantiate(id: string, data: PipelineInstantiateReq): Promise<PipelineResp> {
    return request.post(`/api/pipeline/${id}/instantiate`, data);
  },

  importStage(id: string, data: PipelineStageImportReq): Promise<PipelineResp> {
    return request.post(`/api/pipeline/${id}/stage`, data);
  },

  updateStage(
    pipelineId: string,
    stageId: string,
    data: PipelineStageNodeUpdateReq
  ): Promise<PipelineResp> {
    return request.put(`/api/pipeline/${pipelineId}/stage/${stageId}`, data);
  },

  deleteStage(pipelineId: string, stageId: string): Promise<PipelineResp> {
    return request.delete(`/api/pipeline/${pipelineId}/stage/${stageId}`);
  },

  previewStageTemplateUpdate(
    pipelineId: string,
    stageId: string
  ): Promise<PipelineStageTemplateUpdatePreviewResp> {
    return request.get(`/api/pipeline/${pipelineId}/stage/${stageId}/template-update-preview`);
  },

  updateStageTemplate(
    pipelineId: string,
    stageId: string,
    data: PipelineStageTemplateUpdateReq
  ): Promise<PipelineResp> {
    return request.post(`/api/pipeline/${pipelineId}/stage/${stageId}/template-update`, data);
  },

  getSnapshot(id: string): Promise<PipelineSnapshotResp> {
    return request.get(`/api/pipeline/snapshot/${id}`);
  },
};
