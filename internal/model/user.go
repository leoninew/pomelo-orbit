package model

import "time"

type User struct {
	Id              string     `db:"id"`
	Username        string     `db:"username"`
	PasswordHash    string     `db:"password_hash"`
	Status          string     `db:"status"`
	OAuthProvider   string     `db:"oauth_provider"`
	OAuthProviderId string     `db:"oauth_provider_id"`
	Email           string     `db:"email"`
	AuthSource      string     `db:"auth_source"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
	LastLoginAt     *time.Time `db:"last_login_at"`
}

type Role struct {
	Id          string    `db:"id"`
	Code        string    `db:"code"`
	Name        string    `db:"name"`
	Description *string   `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type Permission struct {
	Id          string    `db:"id"`
	Code        string    `db:"code"`
	Name        string    `db:"name"`
	Description *string   `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type UserRole struct {
	UserId    string    `db:"user_id"`
	RoleId    string    `db:"role_id"`
	CreatedAt time.Time `db:"created_at"`
}

type RolePermission struct {
	RoleId       string    `db:"role_id"`
	PermissionId string    `db:"permission_id"`
	CreatedAt    time.Time `db:"created_at"`
}

type LoginHistory struct {
	Id        string    `db:"id"`
	UserId    string    `db:"user_id"`
	Username  string    `db:"username"`
	IpAddress *string   `db:"ip_address"`
	UserAgent *string   `db:"user_agent"`
	LoginAt   time.Time `db:"login_at"`
	Success   bool      `db:"success"`
}

// MCPAccessToken is a user-owned credential for the local stdio MCP process.
// TokenHash is never exposed outside the authentication persistence boundary.
type MCPAccessToken struct {
	Id        string     `db:"id"`
	UserId    string     `db:"user_id"`
	Name      string     `db:"name"`
	TokenHash string     `db:"token_hash"`
	ExpiresAt *time.Time `db:"expires_at"`
	CreatedAt time.Time  `db:"created_at"`
}
