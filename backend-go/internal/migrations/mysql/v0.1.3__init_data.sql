-- v0.1.3: Initial auth and project data aligned with backend v0.8.1

INSERT IGNORE INTO user (
    id, username, password_hash, created_at, updated_at, last_login_at,
    oauth_provider, oauth_provider_id, email, auth_source, status
) VALUES (
    '01KKX2YNPF6VJ9N7QYCWG61KVK',
    'admin',
    '$2b$12$/lPw6VtrhLqQqlhm/fbay.yAXhxj.Ru4qIJapLWQGlRzPz8gmE1vW',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z',
    NULL,
    '',
    '',
    NULL,
    'password',
    'enabled'
);

INSERT IGNORE INTO project (
    id, name, code, is_active, created_at, updated_at
) VALUES (
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '默认项目',
    'default',
    1,
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO permission (
    id, code, name, description, created_at, updated_at
) VALUES
    ('01KRXJXVPC6MQ75SZPWYZJSSA7', 'user:read', 'View Users', 'View user management', '2024-03-16T00:00:00Z', '2024-03-16T00:00:00Z'),
    ('01KRXJXVPC6MQ75SZPWYZJSSA8', 'user:write', 'Manage Users', 'Create, update, enable, disable and delete users', '2024-03-16T00:00:00Z', '2024-03-16T00:00:00Z'),
    ('01KRXJXVPC6MQ75SZPWYZJSSA9', 'role:read', 'View Roles', 'View role management', '2024-03-16T00:00:00Z', '2024-03-16T00:00:00Z'),
    ('01KRXJXVPC6MQ75SZPWYZJSSAA', 'role:write', 'Manage Roles', 'Create, update and delete roles', '2024-03-16T00:00:00Z', '2024-03-16T00:00:00Z');

INSERT IGNORE INTO role (
    id, code, name, description, created_at, updated_at
) VALUES (
    '01KRXJXVPC6MQ75SZPWYZJSSAB',
    'admin',
    'Admin',
    'System administrator',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO role_permission (role_id, permission_id, created_at) VALUES
    ('01KRXJXVPC6MQ75SZPWYZJSSAB', '01KRXJXVPC6MQ75SZPWYZJSSA7', '2024-03-16T00:00:00Z'),
    ('01KRXJXVPC6MQ75SZPWYZJSSAB', '01KRXJXVPC6MQ75SZPWYZJSSA8', '2024-03-16T00:00:00Z'),
    ('01KRXJXVPC6MQ75SZPWYZJSSAB', '01KRXJXVPC6MQ75SZPWYZJSSA9', '2024-03-16T00:00:00Z'),
    ('01KRXJXVPC6MQ75SZPWYZJSSAB', '01KRXJXVPC6MQ75SZPWYZJSSAA', '2024-03-16T00:00:00Z');

INSERT IGNORE INTO user_role (user_id, role_id, created_at) VALUES (
    '01KKX2YNPF6VJ9N7QYCWG61KVK',
    '01KRXJXVPC6MQ75SZPWYZJSSAB',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO project_member (project_id, user_id, created_at) VALUES (
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '01KKX2YNPF6VJ9N7QYCWG61KVK',
    '2024-03-16T00:00:00Z'
);
