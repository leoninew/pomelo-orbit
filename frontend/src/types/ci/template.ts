// PipelineTemplate
export interface VariableDeclaration {
	name: string
	description?: string
	required: boolean
	default?: string
}

export interface PipelineTemplate {
	id: string
	name: string
	description?: string
	content: string
	variable_declarations: VariableDeclaration[]
	is_builtin: boolean
	created_at: string
	updated_at: string
}

export interface PipelineTemplateCreateReq {
	name: string
	description?: string
	content: string
	variable_declarations?: VariableDeclaration[]
}

export interface PipelineTemplateUpdateReq {
	name?: string
	description?: string
	content?: string
	variable_declarations?: VariableDeclaration[]
}
