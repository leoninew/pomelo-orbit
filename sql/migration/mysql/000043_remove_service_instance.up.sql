ALTER TABLE service
    DROP INDEX uq_service_application_instance,
    DROP COLUMN instance_key;
