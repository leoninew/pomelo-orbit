ALTER TABLE gateway_config
    DROP COLUMN external_domain,
    CHANGE COLUMN internal_domain base_domain VARCHAR(255) NOT NULL;
