ALTER TABLE route
    ADD COLUMN acme_challenge TEXT NOT NULL DEFAULT 'http' CHECK (acme_challenge IN ('http', 'dns'));
