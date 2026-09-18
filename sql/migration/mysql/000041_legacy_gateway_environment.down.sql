DROP INDEX uq_application_project_name ON application;
DROP INDEX uq_application_project_code ON application;
CREATE UNIQUE INDEX name ON application(name);

UPDATE environment
SET gateway_application_id = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE id = project_id;

DELETE FROM environment
WHERE id = project_id
  AND target_type = 'local'
  AND target_revision = 1;
