-- v0.1.5: Add login history and system setting permissions aligned with backend v0.8.2

INSERT INTO permission (id, code, name, description, created_at, updated_at)
SELECT
    '01KRXJXVPC6MQ75SZPWYZJSSAC',
    'login:read',
    'View Login History',
    'View login history',
    datetime('now'),
    datetime('now')
WHERE NOT EXISTS (
    SELECT 1 FROM permission WHERE code = 'login:read'
);

INSERT INTO permission (id, code, name, description, created_at, updated_at)
SELECT
    '01KRXJXVPC6MQ75SZPWYZJSSAD',
    'setting:read',
    'View Settings',
    'View system configuration',
    datetime('now'),
    datetime('now')
WHERE NOT EXISTS (
    SELECT 1 FROM permission WHERE code = 'setting:read'
);

INSERT INTO permission (id, code, name, description, created_at, updated_at)
SELECT
    '01KRXJXVPC6MQ75SZPWYZJSSAE',
    'setting:write',
    'Manage Settings',
    'Update and reset system configuration',
    datetime('now'),
    datetime('now')
WHERE NOT EXISTS (
    SELECT 1 FROM permission WHERE code = 'setting:write'
);

INSERT INTO role_permission (role_id, permission_id, created_at)
SELECT role.id, permission.id, datetime('now')
FROM role
JOIN permission ON permission.code = 'login:read'
WHERE role.code = 'admin'
  AND NOT EXISTS (
      SELECT 1
      FROM role_permission
      WHERE role_permission.role_id = role.id
        AND role_permission.permission_id = permission.id
  );

INSERT INTO role_permission (role_id, permission_id, created_at)
SELECT role.id, permission.id, datetime('now')
FROM role
JOIN permission ON permission.code = 'setting:read'
WHERE role.code = 'admin'
  AND NOT EXISTS (
      SELECT 1
      FROM role_permission
      WHERE role_permission.role_id = role.id
        AND role_permission.permission_id = permission.id
  );

INSERT INTO role_permission (role_id, permission_id, created_at)
SELECT role.id, permission.id, datetime('now')
FROM role
JOIN permission ON permission.code = 'setting:write'
WHERE role.code = 'admin'
  AND NOT EXISTS (
      SELECT 1
      FROM role_permission
      WHERE role_permission.role_id = role.id
        AND role_permission.permission_id = permission.id
  );
