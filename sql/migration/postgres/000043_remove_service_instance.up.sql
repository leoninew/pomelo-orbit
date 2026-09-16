ALTER TABLE service
    DROP CONSTRAINT IF EXISTS service_application_id_instance_key_key,
    DROP COLUMN instance_key;
