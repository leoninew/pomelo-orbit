// Credential
export interface Credential {
	id: string
	name: string
	type: string
	created_at: string
	updated_at: string
}

export interface CredentialCreateReq {
	name: string
	type: string
	data: string
}

export interface CredentialUpdateReq {
	name?: string
	data?: string
}

export interface CredentialExportResp {
	version: string
	name: string
	type: string
	data: string
}

export interface CredentialImportReq {
	version?: string
	name: string
	type: string
	data: string
}

export const credentialTypeLabels: Record<string, string> = {
	git_ssh: 'Git SSH',
	git_token: 'Git Token',
	registry_token: 'Registry Token',
};
