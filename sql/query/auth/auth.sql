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
