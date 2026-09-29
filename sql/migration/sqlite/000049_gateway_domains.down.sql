ALTER TABLE gateway_config DROP COLUMN external_domain;
ALTER TABLE gateway_config RENAME COLUMN internal_domain TO base_domain;
