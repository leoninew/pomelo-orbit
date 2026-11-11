ALTER TABLE environment ADD COLUMN host_identity_policy TEXT NOT NULL DEFAULT 'negotiated' CHECK (host_identity_policy IN ('negotiated', 'stable'));
