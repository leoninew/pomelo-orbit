UPDATE environment e
SET gateway_application_id = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE e.id = e.project_id;

DELETE FROM environment
WHERE id = project_id
  AND target_type = 'local'
  AND target_revision = 1;
