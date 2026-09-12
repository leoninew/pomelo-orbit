-- Adopt the Gateway bundle created by the pre-Environment schema.
-- The target is intentionally unprobed and has no workspace root; the
-- Project Initialization Wizard must collect and probe those values.
INSERT INTO environment (
    id, project_id, code, state, target_type, target_revision,
    gateway_application_id
)
SELECT p.id, p.id, p.code, 'active', 'local', 1, a.id
FROM project p
JOIN application a ON a.project_id = p.id
JOIN gateway_config gc ON gc.application_id = a.id
WHERE a.name = 'Traefik'
  AND a.code = 'traefik'
  AND EXISTS (
      SELECT 1 FROM service s
      WHERE s.application_id = a.id AND s.instance_key = 'default'
  )
  AND NOT EXISTS (
      SELECT 1 FROM environment e
      WHERE e.project_id = p.id OR e.id = p.id OR e.gateway_application_id = a.id
  );

UPDATE environment e
JOIN application a ON a.project_id = e.project_id
JOIN gateway_config gc ON gc.application_id = a.id
SET e.gateway_application_id = a.id,
    e.updated_at = CURRENT_TIMESTAMP
WHERE e.gateway_application_id IS NULL
  AND a.name = 'Traefik'
  AND a.code = 'traefik'
  AND EXISTS (
      SELECT 1 FROM service s
      WHERE s.application_id = a.id AND s.instance_key = 'default'
  )
  AND NOT EXISTS (
      SELECT 1 FROM environment other
      WHERE other.gateway_application_id = a.id AND other.id <> e.id
  );
