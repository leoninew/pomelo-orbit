-- identity seed captured from data/mysql-transfer-20260816-103803.mysql.sql.
-- Upserts support databases previously initialized by the removed data-migration loader.

-- user: 1 row(s).
INSERT INTO `user` (`id`, `username`, `password_hash`, `last_login_at`, `oauth_provider`, `oauth_provider_id`, `email`, `auth_source`, `status`) VALUES
    ('01KKX2YNPF6VJ9N7QYCWG61KVK', 'admin', '$2b$12$/lPw6VtrhLqQqlhm/fbay.yAXhxj.Ru4qIJapLWQGlRzPz8gmE1vW', NULL, '', '', NULL, 'password', 'enabled')
ON DUPLICATE KEY UPDATE
    `username` = VALUES(`username`),
    `password_hash` = VALUES(`password_hash`),
    `last_login_at` = VALUES(`last_login_at`),
    `oauth_provider` = VALUES(`oauth_provider`),
    `oauth_provider_id` = VALUES(`oauth_provider_id`),
    `email` = VALUES(`email`),
    `auth_source` = VALUES(`auth_source`),
    `status` = VALUES(`status`);

-- project: 1 row(s).
INSERT INTO `project` (`id`, `name`, `code`, `is_active`) VALUES
    ('01KRRKK0K3T519ZQZES3M4QA9Z', '默认项目', 'default', 1)
ON DUPLICATE KEY UPDATE
    `name` = VALUES(`name`),
    `code` = VALUES(`code`),
    `is_active` = VALUES(`is_active`);

-- permission: 9 row(s).
INSERT INTO `permission` (`id`, `code`, `name`, `description`) VALUES
    ('01KRXJXVPC6MQ75SZPWYZJSSA7', 'user:read', 'View Users', 'View user management'),
    ('01KRXJXVPC6MQ75SZPWYZJSSA8', 'user:write', 'Manage Users', 'Create, update, enable, disable and delete users'),
    ('01KRXJXVPC6MQ75SZPWYZJSSA9', 'role:read', 'View Roles', 'View role management'),
    ('01KRXJXVPC6MQ75SZPWYZJSSAA', 'role:write', 'Manage Roles', 'Create, update and delete roles'),
    ('01KRXJXVPC6MQ75SZPWYZJSSAC', 'login:read', 'View Login History', 'View login history'),
    ('01KRXJXVPC6MQ75SZPWYZJSSAD', 'setting:read', 'View Settings', 'View system configuration'),
    ('01KRXJXVPC6MQ75SZPWYZJSSAE', 'setting:write', 'Manage Settings', 'Update and reset system configuration'),
    ('01KZCQ5XSK2VC5BMC3M0QJTX9A', 'dialogue:read', 'View Deployment Dialogues', 'View deployment dialogue history'),
    ('01KZCQ5XSK2VC5BMC3M0QJTX9B', 'dialogue:write', 'Manage Deployment Dialogues', 'Create, continue and delete deployment dialogues')
ON DUPLICATE KEY UPDATE
    `code` = VALUES(`code`),
    `name` = VALUES(`name`),
    `description` = VALUES(`description`);

-- role: 1 row(s).
INSERT INTO `role` (`id`, `code`, `name`, `description`) VALUES
    ('01KRXJXVPC6MQ75SZPWYZJSSAB', 'admin', 'Admin', 'System administrator')
ON DUPLICATE KEY UPDATE
    `code` = VALUES(`code`),
    `name` = VALUES(`name`),
    `description` = VALUES(`description`);

-- project_member: 1 row(s).
INSERT IGNORE INTO `project_member` (`project_id`, `user_id`) VALUES
    ('01KRRKK0K3T519ZQZES3M4QA9Z', '01KKX2YNPF6VJ9N7QYCWG61KVK');

-- role_permission: 9 row(s).
INSERT IGNORE INTO `role_permission` (`role_id`, `permission_id`) VALUES
    ('01KRXJXVPC6MQ75SZPWYZJSSAB', '01KRXJXVPC6MQ75SZPWYZJSSA7'),
    ('01KRXJXVPC6MQ75SZPWYZJSSAB', '01KRXJXVPC6MQ75SZPWYZJSSA8'),
    ('01KRXJXVPC6MQ75SZPWYZJSSAB', '01KRXJXVPC6MQ75SZPWYZJSSA9'),
    ('01KRXJXVPC6MQ75SZPWYZJSSAB', '01KRXJXVPC6MQ75SZPWYZJSSAA'),
    ('01KRXJXVPC6MQ75SZPWYZJSSAB', '01KRXJXVPC6MQ75SZPWYZJSSAC'),
    ('01KRXJXVPC6MQ75SZPWYZJSSAB', '01KRXJXVPC6MQ75SZPWYZJSSAD'),
    ('01KRXJXVPC6MQ75SZPWYZJSSAB', '01KRXJXVPC6MQ75SZPWYZJSSAE'),
    ('01KRXJXVPC6MQ75SZPWYZJSSAB', '01KZCQ5XSK2VC5BMC3M0QJTX9A'),
    ('01KRXJXVPC6MQ75SZPWYZJSSAB', '01KZCQ5XSK2VC5BMC3M0QJTX9B');

-- user_role: 1 row(s).
INSERT IGNORE INTO `user_role` (`user_id`, `role_id`) VALUES
    ('01KKX2YNPF6VJ9N7QYCWG61KVK', '01KRXJXVPC6MQ75SZPWYZJSSAB');
