// Stage 配置类型
export interface CheckoutConfig {
	ref: string
}

export interface DockerBuildConfig {
	context: string
	dockerfile: string
	image_name: string
}

export interface UnitTestConfig {
	image: string
	commands: string[]
	artifact_paths: string[]
}

export type StageType = 'checkout' | 'docker_build' | 'unit_test' | 'custom';

export interface StageDefinition {
	name: string
	type: StageType
	depends_on: string[]
	config?: CheckoutConfig | DockerBuildConfig | UnitTestConfig
	steps?: Record<string, unknown>[] // custom 类型的 StepDefinition
}

// 变量声明
export interface VariableDeclaration {
	name: string
	description?: string
	required: boolean
	default?: string
	secret: boolean
	locked: boolean
}

// 流水线模板
export interface PipelineTemplate {
	id: string
	name: string
	description?: string
	stages: StageDefinition[]
	variable_declarations: VariableDeclaration[]
	is_builtin: boolean
	latest_snapshot_version?: number
	created_at: string
	updated_at: string
}

export interface PipelineTemplateCreateReq {
	name: string
	description?: string
	stages?: StageDefinition[]
	variable_declarations?: VariableDeclaration[]
}

export interface PipelineTemplateUpdateReq {
	name?: string
	description?: string
	stages?: StageDefinition[]
	variable_declarations?: VariableDeclaration[]
}
