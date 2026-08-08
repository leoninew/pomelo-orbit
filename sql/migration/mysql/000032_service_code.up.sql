-- Domain: service
ALTER TABLE service
    ADD COLUMN code VARCHAR(63) NULL AFTER instance_key;

UPDATE service AS s
INNER JOIN application AS a ON a.id = s.application_id
SET s.code = CONCAT(a.code, '-', s.instance_key);

ALTER TABLE service
    MODIFY COLUMN code VARCHAR(63) NOT NULL,
    ADD UNIQUE KEY uq_service_code (code);
