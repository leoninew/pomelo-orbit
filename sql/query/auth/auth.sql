-- name: SaveLoginHistory :exec
INSERT INTO login_history (id, user_id, username, ip_address, user_agent, login_at, success)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: CountLoginHistory :one
SELECT COUNT(*)
FROM login_history
WHERE (CAST(sqlc.narg(search_pattern) AS TEXT) IS NULL
  OR LOWER(username) LIKE CAST(sqlc.narg(search_pattern) AS TEXT));

-- name: ListLoginHistory :many
SELECT id, user_id, username, ip_address, user_agent, login_at, success
FROM login_history
WHERE (CAST(sqlc.narg(search_pattern) AS TEXT) IS NULL
  OR LOWER(username) LIKE CAST(sqlc.narg(search_pattern) AS TEXT))
ORDER BY login_at DESC, id DESC
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);
