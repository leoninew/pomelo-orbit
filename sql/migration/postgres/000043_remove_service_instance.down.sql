ALTER TABLE service
    ADD COLUMN instance_key TEXT;

UPDATE service
SET instance_key = code;

ALTER TABLE service
    ALTER COLUMN instance_key SET NOT NULL,
    ADD CONSTRAINT service_application_id_instance_key_key UNIQUE (application_id, instance_key);
