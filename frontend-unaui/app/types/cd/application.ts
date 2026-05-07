export interface Application {
  id: string
  name: string
  code: string
  repository_url: string
  git_credential_id?: string | null
  default_branch: string
  created_at: string
  updated_at: string
}
