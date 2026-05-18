-- v0.7.4: Permission management schema

CREATE TABLE IF NOT EXISTS permission (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS user_role (
    user_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES role(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS role_permission (
    role_id TEXT NOT NULL,
    permission_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES role(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permission(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_permission_code ON permission(code);
CREATE INDEX IF NOT EXISTS idx_user_role_user_id ON user_role(user_id);
CREATE INDEX IF NOT EXISTS idx_user_role_role_id ON user_role(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permission_role_id ON role_permission(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permission_permission_id ON role_permission(permission_id);

INSERT INTO permission (id, code, name, description)
SELECT 'perm-user-read', 'user:read', 'View Users', 'View user management'
WHERE NOT EXISTS (SELECT 1 FROM permission WHERE code = 'user:read');

INSERT INTO permission (id, code, name, description)
SELECT 'perm-user-write', 'user:write', 'Manage Users', 'Create, update, enable, disable and delete users'
WHERE NOT EXISTS (SELECT 1 FROM permission WHERE code = 'user:write');

INSERT INTO permission (id, code, name, description)
SELECT 'perm-role-read', 'role:read', 'View Roles', 'View role management'
WHERE NOT EXISTS (SELECT 1 FROM permission WHERE code = 'role:read');

INSERT INTO permission (id, code, name, description)
SELECT 'perm-role-write', 'role:write', 'Manage Roles', 'Create, update and delete roles'
WHERE NOT EXISTS (SELECT 1 FROM permission WHERE code = 'role:write');

INSERT INTO role (id, code, name, description, is_active)
SELECT 'role-admin', 'admin', 'Admin', 'System administrator', 1
WHERE NOT EXISTS (SELECT 1 FROM role WHERE code = 'admin');

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM role r
JOIN permission p ON p.code IN ('user:read', 'user:write', 'role:read', 'role:write')
WHERE r.code = 'admin'
  AND NOT EXISTS (
      SELECT 1 FROM role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );

INSERT INTO user_role (user_id, role_id)
SELECT u.id, r.id
FROM user u
JOIN role r ON r.code = 'admin'
WHERE u.username = 'admin'
  AND NOT EXISTS (
      SELECT 1 FROM user_role ur WHERE ur.user_id = u.id AND ur.role_id = r.id
  );
