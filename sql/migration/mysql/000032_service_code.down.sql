-- Domain: service — reverse
ALTER TABLE service
    DROP INDEX uq_service_code,
    DROP COLUMN code;
