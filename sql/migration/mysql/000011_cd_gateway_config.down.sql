ALTER TABLE environment
    ADD COLUMN base_domain VARCHAR(255) NOT NULL DEFAULT 'local.test',
    ADD COLUMN domain_template VARCHAR(512) NULL;

DROP TABLE IF EXISTS gateway_config;
