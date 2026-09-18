ALTER TABLE service
    ADD COLUMN instance_key VARCHAR(100) NULL AFTER application_id;

UPDATE service
SET instance_key = code;

ALTER TABLE service
    MODIFY COLUMN instance_key VARCHAR(100) NOT NULL,
    ADD UNIQUE KEY uq_service_application_instance (application_id, instance_key);
