export interface LoginReq {
  username: string
  password: string
}

export interface TokenResp {
  access_token: string
  token_type: string
}

export interface UserInfo {
  id: string
  username: string
  created_at: string
  last_login_at: string | null
}
