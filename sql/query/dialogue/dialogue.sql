-- name: ListDeploymentDialogueConversations :many
SELECT id, project_id, created_by_user_id, title, created_at, updated_at
FROM deployment_dialogue_conversation
WHERE project_id = sqlc.arg(project_id)
ORDER BY updated_at DESC, id DESC;

-- name: DeploymentDialogueConversationById :one
SELECT id, project_id, created_by_user_id, title, created_at, updated_at
FROM deployment_dialogue_conversation
WHERE id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);

-- name: ListDeploymentDialogueMessages :many
SELECT id, conversation_id, role, content, created_at
FROM deployment_dialogue_message
WHERE conversation_id = sqlc.arg(conversation_id)
  AND EXISTS (
    SELECT 1
    FROM deployment_dialogue_conversation
    WHERE id = deployment_dialogue_message.conversation_id
      AND project_id = sqlc.arg(project_id)
  )
ORDER BY created_at ASC, id ASC;

-- name: CreateDeploymentDialogueConversation :exec
INSERT INTO deployment_dialogue_conversation (id, project_id, created_by_user_id, title, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: CreateDeploymentDialogueMessage :execrows
INSERT INTO deployment_dialogue_message (id, conversation_id, role, content, created_at)
SELECT sqlc.arg(id), sqlc.arg(conversation_id), sqlc.arg(role), sqlc.arg(content), sqlc.arg(created_at)
WHERE EXISTS (
  SELECT 1
  FROM deployment_dialogue_conversation AS conversation
  WHERE conversation.id = sqlc.arg(conversation_id)
    AND conversation.project_id = sqlc.arg(project_id)
);

-- name: TouchDeploymentDialogueConversation :exec
UPDATE deployment_dialogue_conversation
SET updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);

-- name: DeleteDeploymentDialogueConversation :exec
DELETE FROM deployment_dialogue_conversation
WHERE id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);
