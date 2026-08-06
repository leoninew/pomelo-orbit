package dto

import "time"

// MCPGrantInput is the browser-side request that binds an authorization code
// to the local loopback callback created by the stdio MCP process.
type MCPGrantInput struct {
	CallbackURL string
	State       string
}

type MCPGrant struct {
	Code      string
	ExpiresAt time.Time
}
