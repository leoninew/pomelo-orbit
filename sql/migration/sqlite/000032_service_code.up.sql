-- Domain: service
-- Service code is the immutable public routing identity.
-- SQLite requires a default when adding a non-null column to an existing table;
-- every existing row is immediately backfilled and runtime inserts always set it.
ALTER TABLE service ADD COLUMN code TEXT NOT NULL DEFAULT '';

UPDATE service
SET code = (
    SELECT application.code || '-' || service.instance_key
    FROM application
    WHERE application.id = service.application_id
);

CREATE UNIQUE INDEX uq_service_code ON service(code);
