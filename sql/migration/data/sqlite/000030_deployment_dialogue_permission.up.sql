INSERT OR IGNORE INTO permission (id, code, name, description, created_at, updated_at) VALUES
    ('01KZCQ5XSK2VC5BMC3M0QJTX9A', 'dialogue:read', 'View Deployment Dialogues', 'View deployment dialogue history', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('01KZCQ5XSK2VC5BMC3M0QJTX9B', 'dialogue:write', 'Manage Deployment Dialogues', 'Create, continue and delete deployment dialogues', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO role_permission (role_id, permission_id, created_at) VALUES
    ('01KRXJXVPC6MQ75SZPWYZJSSAB', '01KZCQ5XSK2VC5BMC3M0QJTX9A', CURRENT_TIMESTAMP),
    ('01KRXJXVPC6MQ75SZPWYZJSSAB', '01KZCQ5XSK2VC5BMC3M0QJTX9B', CURRENT_TIMESTAMP);
