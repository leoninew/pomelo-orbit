-- name: SaveLoginHistory :exec
INSERT INTO login_history (id, user_id, username, ip_address, user_agent, login_at, success)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: CountLoginHistory :one
SELECT COUNT(*)
FROM login_history
WHERE (? = '' OR LOWER(username) LIKE ?);

-- name: ListLoginHistory :many
SELECT id, user_id, username, ip_address, user_agent, login_at, success
FROM login_history
WHERE (? = '' OR LOWER(username) LIKE ?)
ORDER BY login_at DESC, id DESC
LIMIT ? OFFSET ?;
