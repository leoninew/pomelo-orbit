ALTER TABLE route
    ADD COLUMN acme_challenge VARCHAR(16) NOT NULL DEFAULT 'http' CHECK (acme_challenge IN ('http', 'dns'));
