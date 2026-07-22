-- Environment IngressPolicy columns; drop environment_binding.

ALTER TABLE environment ADD COLUMN base_domain TEXT NOT NULL DEFAULT 'local.test';
ALTER TABLE environment ADD COLUMN domain_template TEXT;
ALTER TABLE environment ADD COLUMN default_entrypoint TEXT NOT NULL DEFAULT 'web';
ALTER TABLE environment ADD COLUMN tcp_entrypoint TEXT;
ALTER TABLE environment ADD COLUMN tls_mode TEXT NOT NULL DEFAULT 'none';

UPDATE environment
SET
    base_domain = COALESCE(NULLIF(TRIM(base_domain), ''), 'local.test'),
    default_entrypoint = COALESCE(NULLIF(TRIM(default_entrypoint), ''), 'web'),
    tls_mode = COALESCE(NULLIF(TRIM(tls_mode), ''), 'none');

DROP TABLE IF EXISTS environment_binding;
