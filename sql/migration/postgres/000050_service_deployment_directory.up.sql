ALTER TABLE service ADD COLUMN deployment_directory TEXT NOT NULL DEFAULT '';
ALTER TABLE service ADD COLUMN directory_target_revision BIGINT NOT NULL DEFAULT 0;
ALTER TABLE service ADD COLUMN runtime_directory TEXT NOT NULL DEFAULT '';
ALTER TABLE service ADD COLUMN runtime_target_revision BIGINT NOT NULL DEFAULT 0;
ALTER TABLE deployment ADD COLUMN working_directory TEXT;

UPDATE application SET kind = 'gateway'
WHERE id IN (SELECT application_id FROM gateway_config);

UPDATE service
SET deployment_directory = (SELECT rtrim(replace(e.workspace_root, chr(92), '/'), '/') || '/deployment/' || service.code FROM environment e WHERE e.project_id = service.project_id),
    directory_target_revision = (SELECT e.target_revision FROM environment e WHERE e.project_id = service.project_id)
WHERE EXISTS (SELECT 1 FROM environment e WHERE e.project_id = service.project_id AND e.workspace_root IS NOT NULL AND e.workspace_root <> '');

UPDATE service
SET runtime_directory = deployment_directory, runtime_target_revision = directory_target_revision
WHERE deployment_directory <> '' AND EXISTS (
    SELECT 1 FROM deployment d
    WHERE d.service_id = service.id AND d.environment_target_revision = service.directory_target_revision
      AND d.status IN ('running', 'ran_to_completion')
);

UPDATE deployment
SET working_directory = (SELECT s.deployment_directory FROM service s WHERE s.id = deployment.service_id)
WHERE status IN ('waiting_to_run', 'running') AND EXISTS (
  SELECT 1 FROM service s JOIN environment e ON e.project_id = s.project_id
  WHERE s.id = deployment.service_id AND s.deployment_directory <> ''
    AND e.id = deployment.environment_id
    AND e.target_revision = deployment.environment_target_revision
    AND e.target_type = deployment.environment_target_type
);
