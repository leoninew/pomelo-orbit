PRAGMA foreign_keys = OFF;

CREATE TABLE user_role_new (
    user_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (role_id) REFERENCES role(id) ON DELETE CASCADE
);

INSERT INTO user_role_new (user_id, role_id, created_at)
SELECT user_id, role_id, created_at
FROM user_role;

DROP TABLE user_role;
ALTER TABLE user_role_new RENAME TO user_role;

CREATE INDEX idx_user_role_user_id ON user_role(user_id);
CREATE INDEX idx_user_role_role_id ON user_role(role_id);

PRAGMA foreign_keys = ON;
