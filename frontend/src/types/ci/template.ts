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
	stage_key: string  // 模板内唯一标识，默认为 stage 名，用于 depends_on 引用
	depends_on: string[] // 依赖的 stage_key 列表
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
	required: boolean
	default?: string | null
	secret: boolean
	locked: boolean
}

// ── 模板 ──────────────────────────────────────────────────────────────────────

export interface PipelineTemplate {
	id: string
	name: string
	description: string
	orchestration: StageOrchestration[]
	stages: PipelineStage[] // 编排引用的 Stage 详情
	variable_declarations: VariableDeclaration[]
	latest_snapshot_version?: number
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
	variable_declarations?: VariableDeclaration[]
}
