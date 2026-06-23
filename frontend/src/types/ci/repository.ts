import type { VariableDeclaration } from './template';

export interface RepositoryListItem {
  id: string;
  name: string;
  code: string;
  repository_url: string;
  has_credential: boolean;
  git_credential_id?: string | null;
  default_branch: string;
  created_at: string;
  updated_at: string;
}

export interface Repository {
  id: string;
  name: string;
  code: string;
  repository_url: string;
  has_credential: boolean;
  git_credential_id?: string | null;
  git_credential_name?: string | null;
  variable_declarations: VariableDeclaration[]; // 变量列表（包含内置和自定义）
  default_branch: string;
  created_at: string;
  updated_at: string;
}

export interface RepositoryCreateReq {
  name: string;
  code: string;
  repository_url: string;
  git_credential_id?: string | null;
  variable_overrides?: VariableDeclaration[];
  default_branch?: string;
}

export interface RepositoryUpdateReq {
  name?: string;
  repository_url?: string;
  git_credential_id?: string | null;
  variable_overrides?: VariableDeclaration[];
  default_branch?: string;
}
