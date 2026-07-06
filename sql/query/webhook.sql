-- name: ListRepositoryWebhooks :many
SELECT id, repository_id, name, template_id, branch_filter, encrypted_secret, enabled, created_at, updated_at
FROM repository_webhook
WHERE repository_id = ?
ORDER BY created_at DESC, id;

-- name: RepositoryWebhookByID :one
SELECT id, repository_id, name, template_id, branch_filter, encrypted_secret, enabled, created_at, updated_at
FROM repository_webhook
WHERE id = ?;

-- name: DeleteRepositoryWebhook :exec
DELETE FROM repository_webhook
WHERE id = ?;
