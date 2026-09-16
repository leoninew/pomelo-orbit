ALTER TABLE environment
    ADD COLUMN state TEXT NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'disabled'));
