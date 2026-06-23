import type { TaskStatus } from '../common';
import type { ArtifactType } from './template';

// StageRun — Stage 执行记录
export interface StageRun {
  id: string;
  pipeline_run_id: string;
  stage_id: string;
  stage_name: string;
  status: TaskStatus;
  started_at?: string;
  finished_at?: string;
  exit_code?: number;
  error_message?: string;
}

// Artifact
export interface Artifact {
  id: string;
  pipeline_run_id: string;
  repository_id: string;
  repository_name: string;
  template_id: string;
  template_name: string;
  stage_name: string;
  type: ArtifactType;
  name: string;
  path?: string;
  created_at: string;
}
