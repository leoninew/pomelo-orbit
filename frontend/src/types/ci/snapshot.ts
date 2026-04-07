import type { VariableDeclaration } from './template';

// 快照中的 Stage 定义（执行时展开，含编排属性）
export interface SnapshotStage {
	name: string
	image: string
	depends_on: string[]
	script: string
	env: Record<string, string>
	artifacts?: { path: string; name: string }[]
}

export interface PipelineSnapshotListItem {
	id: string
	template_id: string
	version: number
	created_at: string
}

export interface PipelineSnapshot {
	id: string
	template_id: string
	version: number
	stages_snapshot: SnapshotStage[]
	variable_declarations_snapshot: VariableDeclaration[]
	created_at: string
}
