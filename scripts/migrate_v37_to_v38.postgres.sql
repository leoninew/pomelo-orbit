-- Reconcile a PostgreSQL v37 development database whose version-38 migration
-- was recorded before the project-scoped Environment schema was finalized.
--
-- Run this while connected to the target database with ON_ERROR_STOP enabled.
-- The script leaves schema_migrations unchanged at version 38. Legacy Projects
-- receive a disabled Environment and an exact credential reconfiguration marker;
-- the marker cannot deploy and must be replaced with a real SSH key in the UI.

BEGIN;

DO $$
DECLARE
    migration_version BIGINT;
    migration_dirty BOOLEAN;
    has_project_id BOOLEAN;
    environment_rows BIGINT;
BEGIN
    SELECT version, dirty
    INTO migration_version, migration_dirty
    FROM schema_migrations
    LIMIT 1;

    IF migration_version IS NULL OR migration_version <> 38 OR migration_dirty THEN
        RAISE EXCEPTION 'Expected schema_migrations version 38 with dirty = false';
    END IF;

    SELECT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'environment'
          AND column_name = 'project_id'
    ) INTO has_project_id;

    IF NOT has_project_id THEN
        SELECT COUNT(*) INTO environment_rows FROM environment;
        IF environment_rows <> 0 THEN
            RAISE EXCEPTION 'Cannot infer project_id for existing environment records';
        END IF;
    END IF;
END $$;

DO $$
DECLARE
    foreign_key_name TEXT;
BEGIN
    FOR foreign_key_name IN
        SELECT constraint_name
        FROM information_schema.table_constraints
        WHERE table_schema = current_schema()
          AND table_name = 'environment'
          AND constraint_type = 'FOREIGN KEY'
    LOOP
        EXECUTE format('ALTER TABLE environment DROP CONSTRAINT %I', foreign_key_name);
    END LOOP;
END $$;

ALTER TABLE environment DROP COLUMN IF EXISTS name;

ALTER TABLE deployment
    ADD COLUMN IF NOT EXISTS environment_id TEXT,
    ADD COLUMN IF NOT EXISTS environment_target_revision BIGINT,
    ADD COLUMN IF NOT EXISTS ssh_credential_id TEXT,
    ADD COLUMN IF NOT EXISTS ssh_credential_revision BIGINT,
    ADD COLUMN IF NOT EXISTS gateway_application_id TEXT;

ALTER TABLE credential
    ADD COLUMN IF NOT EXISTS revision BIGINT NOT NULL DEFAULT 1;

ALTER TABLE environment
    ADD COLUMN IF NOT EXISTS project_id TEXT NOT NULL;

DO $$
DECLARE
    project_id_attribute SMALLINT;
    has_project_unique_constraint BOOLEAN;
BEGIN
    SELECT attribute.attnum
    INTO project_id_attribute
    FROM pg_attribute AS attribute
    JOIN pg_class AS relation ON relation.oid = attribute.attrelid
    JOIN pg_namespace AS namespace ON namespace.oid = relation.relnamespace
    WHERE namespace.nspname = current_schema()
      AND relation.relname = 'environment'
      AND attribute.attname = 'project_id'
      AND attribute.attnum > 0
      AND NOT attribute.attisdropped;

    SELECT EXISTS (
        SELECT 1
        FROM pg_constraint AS constraint
        JOIN pg_class AS relation ON relation.oid = constraint.conrelid
        JOIN pg_namespace AS namespace ON namespace.oid = relation.relnamespace
        WHERE namespace.nspname = current_schema()
          AND relation.relname = 'environment'
          AND constraint.contype = 'u'
          AND array_length(constraint.conkey, 1) = 1
          AND constraint.conkey[1] = project_id_attribute
    ) INTO has_project_unique_constraint;

    IF NOT has_project_unique_constraint THEN
        ALTER TABLE environment ADD CONSTRAINT uq_environment_project_id UNIQUE (project_id);
    END IF;
END $$;

DO $$
DECLARE
    placeholder_conflicts BIGINT;
    code_conflicts BIGINT;
BEGIN
    SELECT COUNT(*) INTO placeholder_conflicts
    FROM project AS p
    LEFT JOIN environment AS e ON e.project_id = p.id
    INNER JOIN credential AS c ON c.id = 'MIG' || UPPER(SUBSTRING(MD5('pomelo-orbit-v37-to-v38-credential:' || p.id) FROM 1 FOR 23))
    WHERE e.id IS NULL
      AND (
          c.project_id IS NULL
          OR c.project_id <> p.id
          OR c.type <> 'deployment_ssh_private_key'
          OR c.encrypted_data <> '__POMELO_ORBIT_DEPLOYMENT_SSH_RECONFIGURATION_REQUIRED__'
      );

    IF placeholder_conflicts <> 0 THEN
        RAISE EXCEPTION 'Legacy environment credential placeholder conflicts with existing data';
    END IF;

    SELECT COUNT(*) INTO code_conflicts
    FROM project AS p
    LEFT JOIN environment AS own_environment ON own_environment.project_id = p.id
    INNER JOIN environment AS other_environment
        ON other_environment.code = p.code
       AND other_environment.project_id <> p.id
    WHERE own_environment.id IS NULL;

    IF code_conflicts <> 0 THEN
        RAISE EXCEPTION 'Legacy project code conflicts with an existing environment code';
    END IF;

    INSERT INTO credential (id, project_id, name, type, encrypted_data, revision, created_at)
    SELECT
        'MIG' || UPPER(SUBSTRING(MD5('pomelo-orbit-v37-to-v38-credential:' || p.id) FROM 1 FOR 23)),
        p.id,
        'deployment-ssh-reconfiguration',
        'deployment_ssh_private_key',
        '__POMELO_ORBIT_DEPLOYMENT_SSH_RECONFIGURATION_REQUIRED__',
        1,
        CURRENT_TIMESTAMP
    FROM project AS p
    LEFT JOIN environment AS e ON e.project_id = p.id
    LEFT JOIN credential AS c ON c.id = 'MIG' || UPPER(SUBSTRING(MD5('pomelo-orbit-v37-to-v38-credential:' || p.id) FROM 1 FOR 23))
    WHERE e.id IS NULL
      AND c.id IS NULL;

    INSERT INTO environment (
        id,
        project_id,
        code,
        state,
        platform,
        host,
        port,
        username,
        workspace_root,
        ssh_credential_id,
        ssh_credential_revision,
        host_key_fingerprint,
        target_revision,
        created_at,
        updated_at
    )
    SELECT
        'ENV' || UPPER(SUBSTRING(MD5('pomelo-orbit-v37-to-v38-environment:' || p.id) FROM 1 FOR 23)),
        p.id,
        p.code,
        'disabled',
        'linux',
        'unconfigured.invalid',
        22,
        'unconfigured',
        '/var/lib/pomelo-orbit',
        c.id,
        c.revision,
        'SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=',
        1,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    FROM project AS p
    LEFT JOIN environment AS e ON e.project_id = p.id
    INNER JOIN credential AS c ON c.id = 'MIG' || UPPER(SUBSTRING(MD5('pomelo-orbit-v37-to-v38-credential:' || p.id) FROM 1 FOR 23))
    WHERE e.id IS NULL;
END $$;

DO $$
DECLARE
    ambiguous_environment_count BIGINT;
BEGIN
    SELECT COUNT(*) INTO ambiguous_environment_count
    FROM (
        SELECT e.id
        FROM environment AS e
        INNER JOIN application AS a ON a.project_id = e.project_id
        INNER JOIN gateway_config AS gc ON gc.application_id = a.id
        WHERE e.gateway_application_id IS NULL
        GROUP BY e.id
        HAVING COUNT(*) > 1
    ) AS ambiguous_environments;

    IF ambiguous_environment_count <> 0 THEN
        RAISE EXCEPTION 'Legacy project has multiple configured gateways; environment binding is ambiguous';
    END IF;

    UPDATE environment AS e
    SET gateway_application_id = candidate.application_id,
        updated_at = CURRENT_TIMESTAMP
    FROM (
        SELECT a.project_id, MIN(a.id) AS application_id
        FROM application AS a
        INNER JOIN gateway_config AS gc ON gc.application_id = a.id
        GROUP BY a.project_id
        HAVING COUNT(*) = 1
    ) AS candidate
    WHERE candidate.project_id = e.project_id
      AND e.gateway_application_id IS NULL;
END $$;

DO $$
DECLARE
    deployment_columns INT;
    credential_columns INT;
    environment_project_columns INT;
    environment_project_unique_constraints INT;
    environment_name_columns INT;
    environment_foreign_keys INT;
    projects_without_environment INT;
    invalid_environment_credentials INT;
    invalid_environment_gateway_bindings INT;
BEGIN
    SELECT COUNT(*) INTO deployment_columns
    FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'deployment'
      AND column_name IN (
          'environment_id',
          'environment_target_revision',
          'ssh_credential_id',
          'ssh_credential_revision',
          'gateway_application_id'
      );

    SELECT COUNT(*) INTO credential_columns
    FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'credential'
      AND column_name = 'revision';

    SELECT COUNT(*) INTO environment_project_columns
    FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'environment'
      AND column_name = 'project_id';

    SELECT COUNT(*) INTO environment_project_unique_constraints
    FROM pg_constraint AS constraint
    JOIN pg_class AS relation ON relation.oid = constraint.conrelid
    JOIN pg_namespace AS namespace ON namespace.oid = relation.relnamespace
    JOIN pg_attribute AS attribute
        ON attribute.attrelid = relation.oid
       AND attribute.attname = 'project_id'
       AND attribute.attnum > 0
       AND NOT attribute.attisdropped
    WHERE namespace.nspname = current_schema()
      AND relation.relname = 'environment'
      AND constraint.contype = 'u'
      AND array_length(constraint.conkey, 1) = 1
      AND constraint.conkey[1] = attribute.attnum;

    SELECT COUNT(*) INTO environment_name_columns
    FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'environment'
      AND column_name = 'name';

    SELECT COUNT(*) INTO environment_foreign_keys
    FROM information_schema.table_constraints
    WHERE table_schema = current_schema()
      AND table_name = 'environment'
      AND constraint_type = 'FOREIGN KEY';

    SELECT COUNT(*) INTO projects_without_environment
    FROM project AS p
    LEFT JOIN environment AS e ON e.project_id = p.id
    WHERE e.id IS NULL;

    SELECT COUNT(*) INTO invalid_environment_credentials
    FROM environment AS e
    LEFT JOIN credential AS c ON c.id = e.ssh_credential_id
    WHERE c.id IS NULL
       OR c.project_id IS NULL
       OR c.project_id <> e.project_id
       OR c.type <> 'deployment_ssh_private_key'
       OR c.revision <> e.ssh_credential_revision;

    SELECT COUNT(*) INTO invalid_environment_gateway_bindings
    FROM environment AS e
    LEFT JOIN application AS a ON a.id = e.gateway_application_id
    LEFT JOIN gateway_config AS gc ON gc.application_id = a.id
    WHERE e.gateway_application_id IS NOT NULL
      AND (
          a.id IS NULL
          OR a.project_id IS NULL
          OR a.project_id <> e.project_id
          OR gc.application_id IS NULL
      );

    IF deployment_columns <> 5
       OR credential_columns <> 1
       OR environment_project_columns <> 1
       OR environment_project_unique_constraints < 1
       OR environment_name_columns <> 0
       OR environment_foreign_keys <> 0
       OR projects_without_environment <> 0
       OR invalid_environment_credentials <> 0
       OR invalid_environment_gateway_bindings <> 0 THEN
        RAISE EXCEPTION 'v37 to v38 schema and environment verification failed';
    END IF;
END $$;

COMMIT;

SELECT version, dirty FROM schema_migrations;