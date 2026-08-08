-- Domain: service — reverse
DROP INDEX IF EXISTS uq_service_code;
ALTER TABLE service DROP COLUMN code;
