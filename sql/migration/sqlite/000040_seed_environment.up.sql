-- Replaceable default Environment for the seeded Project. Disabled until the
-- SSH target and deploy key are updated in the UI.
INSERT INTO "credential" ("id", "name", "type", "encrypted_data", "project_id", "revision") VALUES
    ('01M202WNXY6FPPFGTJWCF6CP81', 'default-deployment-ssh', 'deployment_ssh_private_key', '__POMELO_ORBIT_DEPLOYMENT_SSH_RECONFIGURATION_REQUIRED__', '01KRRKK0K3T519ZQZES3M4QA9Z', 1);

INSERT INTO "environment" (
    "id", "project_id", "code", "state", "platform", "host", "port", "username", "workspace_root",
    "ssh_credential_id", "ssh_credential_revision", "host_key_fingerprint", "target_revision", "gateway_application_id"
) VALUES (
    '01M202WNXYDEY0E0GDF1RCY3JB', '01KRRKK0K3T519ZQZES3M4QA9Z', 'default', 'disabled', 'linux', '127.0.0.1', 22, 'orbit', '/var/lib/pomelo-orbit',
    '01M202WNXY6FPPFGTJWCF6CP81', 1, 'SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=', 1, '01M10RRA8F863EJ2N9TC3Z2EC1'
);
