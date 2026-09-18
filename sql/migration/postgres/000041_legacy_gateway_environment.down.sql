DROP INDEX IF EXISTS uq_application_project_name;
DROP INDEX IF EXISTS uq_application_project_code;
ALTER TABLE application ADD CONSTRAINT application_name_key UNIQUE (name);

UPDATE environment e
SET gateway_application_id = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE e.id = e.project_id;

DELETE FROM environment
WHERE id = project_id
  AND target_type = 'local'
  AND target_revision = 1;
