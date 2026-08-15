-- name: ListDeploymentDialogueConversations :many
SELECT id, project_id, created_by_user_id, title, created_at, updated_at
FROM deployment_dialogue_conversation
WHERE project_id = ?
ORDER BY updated_at DESC, id DESC;

-- name: DeploymentDialogueConversationByID :one
SELECT id, project_id, created_by_user_id, title, created_at, updated_at
FROM deployment_dialogue_conversation
WHERE id = ?;

-- name: ListDeploymentDialogueMessages :many
SELECT id, conversation_id, role, content, created_at
FROM deployment_dialogue_message
WHERE conversation_id = ?
ORDER BY created_at ASC, id ASC;

-- name: CreateDeploymentDialogueConversation :exec
INSERT INTO deployment_dialogue_conversation (id, project_id, created_by_user_id, title, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: CreateDeploymentDialogueMessage :exec
INSERT INTO deployment_dialogue_message (id, conversation_id, role, content, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: TouchDeploymentDialogueConversation :exec
UPDATE deployment_dialogue_conversation
SET updated_at = ?
WHERE id = ?;

-- name: DeleteDeploymentDialogueConversation :exec
DELETE FROM deployment_dialogue_conversation
WHERE id = ?;
