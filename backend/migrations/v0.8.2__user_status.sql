ALTER TABLE user ADD COLUMN status TEXT NOT NULL DEFAULT 'enabled' CHECK (status IN ('enabled', 'disabled'));

UPDATE user
SET status = CASE WHEN is_active THEN 'enabled' ELSE 'disabled' END;

DROP INDEX IF EXISTS idx_user_is_active;
CREATE INDEX IF NOT EXISTS idx_user_status ON user(status);

ALTER TABLE user DROP COLUMN is_active;

DROP INDEX IF EXISTS idx_role_is_active;

ALTER TABLE role DROP COLUMN is_active;
