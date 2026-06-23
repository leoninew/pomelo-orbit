import type { ArtifactConfig, VariableDeclaration } from './template';

// 快照中的 Stage 定义（执行时展开，含编排属性）
export interface SnapshotStage {
  id: string;
  name: string;
  image: string;
  version: number;
  depends_on: string[]; // 存储依赖的 stage_id 列表
  script: string;
  artifacts?: ArtifactConfig[];
}

export interface PipelineSnapshot {
  id: string;
  template_id: string;
  template_name: string;
  template_version: number;
  version: number;
  stages_snapshot: SnapshotStage[];
  variables_snapshot: VariableDeclaration[];
  created_at: string;
}
