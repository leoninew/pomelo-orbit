-- identity seed captured from data/mysql-transfer-20260816-103803.mysql.sql.

-- user: 1 row(s).
INSERT INTO `user` (`id`, `username`, `password_hash`, `last_login_at`, `oauth_provider`, `oauth_provider_id`, `email`, `auth_source`, `status`) VALUES
    ('01KKX2YNPF6VJ9N7QYCWG61KVK', 'admin', '$2a$10$YZKUsKzahDjcrEpK5bFK.OL6zkm8.Zaf5avFkBo7ma8DuSTXb5ZRu', NULL, '', '', 'admin@lvh.me', 'password', 'enabled');

-- project: 1 row(s).
INSERT INTO `project` (`id`, `name`, `code`, `is_active`) VALUES
    ('01KRRKK0K3T519ZQZES3M4QA9Z', '默认项目', 'default', 1);

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
    ('01KZCQ5XSK2VC5BMC3M0QJTX9B', 'dialogue:write', 'Manage Deployment Dialogues', 'Create, continue and delete deployment dialogues');

-- role: 1 row(s).
INSERT INTO `role` (`id`, `code`, `name`, `description`) VALUES
    ('01KRXJXVPC6MQ75SZPWYZJSSAB', 'admin', 'Admin', 'System administrator');

-- project_member: 1 row(s).
INSERT INTO `project_member` (`project_id`, `user_id`) VALUES
    ('01KRRKK0K3T519ZQZES3M4QA9Z', '01KKX2YNPF6VJ9N7QYCWG61KVK');

-- role_permission: 9 row(s).
INSERT INTO `role_permission` (`role_id`, `permission_id`) VALUES
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
INSERT INTO `user_role` (`user_id`, `role_id`) VALUES
    ('01KKX2YNPF6VJ9N7QYCWG61KVK', '01KRXJXVPC6MQ75SZPWYZJSSAB');
