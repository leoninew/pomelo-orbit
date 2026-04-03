import type { StageDefinition, VariableDeclaration } from './template';

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
	stages_snapshot: StageDefinition[]
	variable_declarations_snapshot: VariableDeclaration[]
	created_at: string
}
