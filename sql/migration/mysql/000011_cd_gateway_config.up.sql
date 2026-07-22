-- E6: gateway_config 1:1 application(kind=gateway); strip domain fields from environment.

CREATE TABLE IF NOT EXISTS gateway_config (
    application_id VARCHAR(26) NOT NULL PRIMARY KEY,
    rest_api_url VARCHAR(512) NOT NULL,
    base_domain VARCHAR(255) NOT NULL,
    image VARCHAR(512) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_gateway_config_application
        FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE environment
    DROP COLUMN base_domain,
    DROP COLUMN domain_template;
