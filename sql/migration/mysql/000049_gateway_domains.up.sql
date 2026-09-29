ALTER TABLE gateway_config
    CHANGE COLUMN base_domain internal_domain VARCHAR(255) NOT NULL,
    ADD COLUMN external_domain VARCHAR(255) NOT NULL DEFAULT '';
