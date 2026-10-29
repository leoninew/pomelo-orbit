-- Hash-only records have no recoverable token; the new value remains NULL.
ALTER TABLE mcp_access_token ADD COLUMN encrypted_token TEXT NULL;
