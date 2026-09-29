ALTER TABLE gateway_config RENAME COLUMN base_domain TO internal_domain;
ALTER TABLE gateway_config ADD COLUMN external_domain TEXT NOT NULL DEFAULT '';
