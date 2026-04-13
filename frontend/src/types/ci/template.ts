export type ArtifactType = 'docker_image' | 'binary';

export interface ArtifactConfig {
	type: ArtifactType
	path: string
	name: string
}

export interface ArtifactDeclaration {
	stageName: string
	type: string
	name: string
	path: string
}

// ── BuildStage ────────────────────────────────────────────────────────────────

export interface BuildStage {
	id: string
	name: string
	image: string
	script: string
	artifacts?: ArtifactConfig[]
	description: string
	version: number
	created_at: string
	updated_at: string
}

export interface BuildStageCreateReq {
	name: string
	image: string
	script: string
	artifacts?: ArtifactConfig[]
	description?: string
}

export interface BuildStageUpdateReq {
	name?: string
	image?: string
	script?: string
	artifacts?: ArtifactConfig[]
	description?: string
}

// ── 编排 ──────────────────────────────────────────────────────────────────────

export interface StageOrchestration {
	stage_id: string
	stage_name: string
	stage_version: number
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
	default?: string | number | boolean | null // 系统/脚本提供的原始默认值
	value?: string | number | boolean | null // 用户的显式覆盖值
	secret: boolean
	editable?: boolean // 是否允许用户修改，由后端根据 source 设置
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
	stages: BuildStage[]
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
