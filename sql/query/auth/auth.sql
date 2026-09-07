-- name: SaveLoginHistory :exec
INSERT INTO login_history (id, user_id, username, ip_address, user_agent, login_at, success)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: CountLoginHistory :one
SELECT COUNT(*)
FROM login_history
WHERE (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL
  OR LOWER(username) LIKE sqlc.narg(search_pattern));

-- name: ListLoginHistory :many
SELECT id, user_id, username, ip_address, user_agent, login_at, success
FROM login_history
WHERE (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL
  OR LOWER(username) LIKE sqlc.narg(search_pattern))
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: CreateMCPAccessToken :exec
INSERT INTO mcp_access_token (id, user_id, name, token_hash, expires_at, created_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: ListMCPAccessTokens :many
SELECT id, user_id, name, token_hash, expires_at, created_at
FROM mcp_access_token
WHERE user_id = ?
ORDER BY created_at DESC, id DESC;

-- name: MCPAccessTokenByHash :one
SELECT id, user_id, name, token_hash, expires_at, created_at
FROM mcp_access_token
WHERE token_hash = ?;

-- name: MCPAccessTokenForUser :one
SELECT id, user_id, name, token_hash, expires_at, created_at
FROM mcp_access_token
WHERE id = ? AND user_id = ?;

-- name: DeleteMCPAccessToken :exec
DELETE FROM mcp_access_token
WHERE id = ?;
