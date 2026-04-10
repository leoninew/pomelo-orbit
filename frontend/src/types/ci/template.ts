export interface ArtifactConfig {
	path: string
	name: string
}

// ── PipelineStage ─────────────────────────────────────────────────────────────

export interface PipelineStage {
	id: string
	name: string
	image: string
	script: string
	env: Record<string, string>
	artifacts?: ArtifactConfig[]
	description: string
	created_at: string
	updated_at: string
}

export interface PipelineStageCreateReq {
	name: string
	image: string
	script: string
	env?: Record<string, string>
	artifacts?: ArtifactConfig[]
	description?: string
}

export interface PipelineStageUpdateReq {
	name?: string
	image?: string
	script?: string
	env?: Record<string, string>
	artifacts?: ArtifactConfig[]
	description?: string
}

// ── 编排 ──────────────────────────────────────────────────────────────────────

export interface StageOrchestration {
	stage_id: string
	stage_name: string
	depends_on: string[] // 存储 stage_id 列表
	sort_order: number
}

export interface OrchestrationUpdateReq {
	orchestration: StageOrchestration[]
	variable_declarations?: VariableDeclaration[]
}

// ── 变量声明 ──────────────────────────────────────────────────────────────────

export interface VariableDeclaration {
	name: string
	description?: string
	value?: string | number | boolean | null
	secret: boolean
	source?:
		| 'global'
		| 'repository'
		| 'repository_custom'
		| 'template'
		| 'template_stage'
		| 'template_custom'
		| 'runtime'
}

// ── 模板 ──────────────────────────────────────────────────────────────────────

export interface PipelineTemplate {
	id: string
	name: string
	description: string
	orchestration: StageOrchestration[]
	stages: PipelineStage[]
	variable_declarations: VariableDeclaration[]
	version: number
	created_at: string
	updated_at: string
}

export interface PipelineTemplateCreateReq {
	name: string
	description?: string
	variable_declarations?: VariableDeclaration[]
}

export interface PipelineTemplateUpdateReq {
	name?: string
	description?: string
	orchestration?: StageOrchestration[]
	variable_declarations?: VariableDeclaration[]
}
