-- Down: drop environment/expose/binding; no application_route rebuild.

DELETE rp FROM role_permission rp
INNER JOIN permission p ON p.id = rp.permission_id
WHERE p.code IN ('environment:read', 'environment:write');

DELETE FROM permission WHERE code IN ('environment:read', 'environment:write');

ALTER TABLE deployment DROP FOREIGN KEY fk_deployment_environment;
ALTER TABLE deployment DROP COLUMN environment_id;

ALTER TABLE service DROP FOREIGN KEY fk_service_environment;
ALTER TABLE service DROP INDEX uq_service_app_env_instance;
ALTER TABLE service DROP INDEX idx_service_environment;
ALTER TABLE service DROP INDEX idx_service_app_env;
ALTER TABLE service DROP COLUMN environment_id;
ALTER TABLE service DROP COLUMN instance_key;
ALTER TABLE service DROP COLUMN is_ingress;
ALTER TABLE service ADD UNIQUE KEY application_id (application_id);

DROP TABLE IF EXISTS environment_binding;
DROP TABLE IF EXISTS expose;
DROP TABLE IF EXISTS environment;
