-- Reconcile a MySQL v37 development database whose version-38 migration was
-- recorded before the project-scoped Environment schema was finalized.
--
-- Run this while connected to the target database. MySQL 8.0+ is required.
-- The script leaves schema_migrations unchanged at version 38. Legacy Projects
-- receive a disabled Environment and an exact credential reconfiguration marker;
-- the marker cannot deploy and must be replaced with a real SSH key in the UI.

DELIMITER //

CREATE PROCEDURE _pomelo_orbit_v37_to_v38_preflight()
BEGIN
    DECLARE migration_version BIGINT DEFAULT NULL;
    DECLARE migration_dirty TINYINT DEFAULT NULL;
    DECLARE has_project_id INT DEFAULT 0;
    DECLARE environment_rows BIGINT DEFAULT 0;

    SELECT version, dirty INTO migration_version, migration_dirty
    FROM schema_migrations
    LIMIT 1;

    IF migration_version IS NULL OR migration_version <> 38 OR migration_dirty <> 0 THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'Expected schema_migrations version 38 with dirty = 0';
    END IF;

    SELECT COUNT(*) INTO has_project_id
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'environment'
      AND column_name = 'project_id';

    IF has_project_id = 0 THEN
        SELECT COUNT(*) INTO environment_rows FROM environment;
        IF environment_rows <> 0 THEN
            SIGNAL SQLSTATE '45000'
                SET MESSAGE_TEXT = 'Cannot infer project_id for existing environment records';
        END IF;
    END IF;
END//

CALL _pomelo_orbit_v37_to_v38_preflight()//
DROP PROCEDURE _pomelo_orbit_v37_to_v38_preflight//

DELIMITER ;

DELIMITER //

CREATE PROCEDURE _pomelo_orbit_v37_to_v38_drop_environment_foreign_keys()
BEGIN
    DECLARE done TINYINT DEFAULT 0;
    DECLARE foreign_key_name VARCHAR(64);
    DECLARE foreign_key_cursor CURSOR FOR
        SELECT constraint_name
        FROM information_schema.table_constraints
        WHERE constraint_schema = DATABASE()
          AND table_name = 'environment'
          AND constraint_type = 'FOREIGN KEY';
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = 1;

    OPEN foreign_key_cursor;
    drop_foreign_key_loop: LOOP
        FETCH foreign_key_cursor INTO foreign_key_name;
        IF done = 1 THEN
            LEAVE drop_foreign_key_loop;
        END IF;
        SET @drop_environment_foreign_key_sql = CONCAT(
            'ALTER TABLE environment DROP FOREIGN KEY `',
            REPLACE(foreign_key_name, '`', '``'),
            '`'
        );
        PREPARE drop_environment_foreign_key FROM @drop_environment_foreign_key_sql;
        EXECUTE drop_environment_foreign_key;
        DEALLOCATE PREPARE drop_environment_foreign_key;
    END LOOP;
    CLOSE foreign_key_cursor;
END//

CALL _pomelo_orbit_v37_to_v38_drop_environment_foreign_keys()//
DROP PROCEDURE _pomelo_orbit_v37_to_v38_drop_environment_foreign_keys//

DELIMITER ;

SET @environment_name_column = (
    SELECT column_name
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'environment'
      AND column_name = 'name'
    LIMIT 1
);
SET @drop_environment_name_column = IF(
    @environment_name_column IS NULL,
    'SELECT 1',
    'ALTER TABLE environment DROP COLUMN `name`'
);
PREPARE drop_environment_name_column FROM @drop_environment_name_column;
EXECUTE drop_environment_name_column;
DEALLOCATE PREPARE drop_environment_name_column;

DELIMITER //

CREATE PROCEDURE _pomelo_orbit_v37_to_v38_ensure_column(
    IN target_table_name VARCHAR(64),
    IN target_column_name VARCHAR(64),
    IN target_column_definition TEXT
)
BEGIN
    DECLARE target_column_count INT DEFAULT 0;

    SELECT COUNT(*) INTO target_column_count
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = target_table_name
      AND column_name = target_column_name;

    IF target_column_count = 0 THEN
        SET @ensure_column_sql = CONCAT(
            'ALTER TABLE `', REPLACE(target_table_name, '`', '``'),
            '` ADD COLUMN `', REPLACE(target_column_name, '`', '``'),
            '` ', target_column_definition
        );
        PREPARE ensure_column_statement FROM @ensure_column_sql;
        EXECUTE ensure_column_statement;
        DEALLOCATE PREPARE ensure_column_statement;
    END IF;
END//

CALL _pomelo_orbit_v37_to_v38_ensure_column('deployment', 'environment_id', 'VARCHAR(26)')//
CALL _pomelo_orbit_v37_to_v38_ensure_column('deployment', 'environment_target_revision', 'BIGINT')//
CALL _pomelo_orbit_v37_to_v38_ensure_column('deployment', 'ssh_credential_id', 'VARCHAR(26)')//
CALL _pomelo_orbit_v37_to_v38_ensure_column('deployment', 'ssh_credential_revision', 'BIGINT')//
CALL _pomelo_orbit_v37_to_v38_ensure_column('deployment', 'gateway_application_id', 'VARCHAR(26)')//
CALL _pomelo_orbit_v37_to_v38_ensure_column('credential', 'revision', 'BIGINT NOT NULL DEFAULT 1')//
CALL _pomelo_orbit_v37_to_v38_ensure_column('environment', 'project_id', 'VARCHAR(26) NOT NULL AFTER `id`')//

DROP PROCEDURE _pomelo_orbit_v37_to_v38_ensure_column//

DELIMITER ;

SET @environment_project_unique_index = (
    SELECT index_name
    FROM (
        SELECT index_name
        FROM information_schema.statistics
        WHERE table_schema = DATABASE()
          AND table_name = 'environment'
          AND non_unique = 0
        GROUP BY index_name
        HAVING COUNT(*) = 1
           AND MIN(column_name) = 'project_id'
    ) AS project_unique_indexes
    LIMIT 1
);
SET @add_environment_project_unique_index = IF(
    @environment_project_unique_index IS NULL,
    'ALTER TABLE environment ADD UNIQUE INDEX uq_environment_project_id (project_id)',
    'SELECT 1'
);
PREPARE add_environment_project_unique_index FROM @add_environment_project_unique_index;
EXECUTE add_environment_project_unique_index;
DEALLOCATE PREPARE add_environment_project_unique_index;

DELIMITER //

CREATE PROCEDURE _pomelo_orbit_v37_to_v38_bootstrap_legacy_projects()
BEGIN
    DECLARE placeholder_conflicts INT DEFAULT 0;
    DECLARE code_conflicts INT DEFAULT 0;

    SELECT COUNT(*) INTO placeholder_conflicts
    FROM project AS p
    LEFT JOIN environment AS e ON e.project_id = p.id
    INNER JOIN credential AS c ON c.id = CONCAT('MIG', UPPER(LEFT(SHA2(CONCAT('pomelo-orbit-v37-to-v38-credential:', p.id), 256), 23)))
    WHERE e.id IS NULL
      AND (
          c.project_id IS NULL
          OR c.project_id <> p.id
          OR c.type <> 'deployment_ssh_private_key'
          OR c.encrypted_data <> '__POMELO_ORBIT_DEPLOYMENT_SSH_RECONFIGURATION_REQUIRED__'
      );

    IF placeholder_conflicts <> 0 THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'Legacy environment credential placeholder conflicts with existing data';
    END IF;

    SELECT COUNT(*) INTO code_conflicts
    FROM project AS p
    LEFT JOIN environment AS own_environment ON own_environment.project_id = p.id
    INNER JOIN environment AS other_environment
        ON other_environment.code = p.code
       AND other_environment.project_id <> p.id
    WHERE own_environment.id IS NULL;

    IF code_conflicts <> 0 THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'Legacy project code conflicts with an existing environment code';
    END IF;

    INSERT INTO credential (id, project_id, name, type, encrypted_data, revision, created_at)
    SELECT
        CONCAT('MIG', UPPER(LEFT(SHA2(CONCAT('pomelo-orbit-v37-to-v38-credential:', p.id), 256), 23))),
        p.id,
        'deployment-ssh-reconfiguration',
        'deployment_ssh_private_key',
        '__POMELO_ORBIT_DEPLOYMENT_SSH_RECONFIGURATION_REQUIRED__',
        1,
        CURRENT_TIMESTAMP(3)
    FROM project AS p
    LEFT JOIN environment AS e ON e.project_id = p.id
    LEFT JOIN credential AS c ON c.id = CONCAT('MIG', UPPER(LEFT(SHA2(CONCAT('pomelo-orbit-v37-to-v38-credential:', p.id), 256), 23)))
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
        CONCAT('ENV', UPPER(LEFT(SHA2(CONCAT('pomelo-orbit-v37-to-v38-environment:', p.id), 256), 23))),
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
    INNER JOIN credential AS c ON c.id = CONCAT('MIG', UPPER(LEFT(SHA2(CONCAT('pomelo-orbit-v37-to-v38-credential:', p.id), 256), 23)))
    WHERE e.id IS NULL;
END//

CALL _pomelo_orbit_v37_to_v38_bootstrap_legacy_projects()//
DROP PROCEDURE _pomelo_orbit_v37_to_v38_bootstrap_legacy_projects//

DELIMITER ;

DELIMITER //

CREATE PROCEDURE _pomelo_orbit_v37_to_v38_bind_legacy_gateways()
BEGIN
    DECLARE ambiguous_environment_count INT DEFAULT 0;

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
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'Legacy project has multiple configured gateways; environment binding is ambiguous';
    END IF;

    UPDATE environment AS e
    INNER JOIN (
        SELECT a.project_id, MIN(a.id) AS application_id
        FROM application AS a
        INNER JOIN gateway_config AS gc ON gc.application_id = a.id
        GROUP BY a.project_id
        HAVING COUNT(*) = 1
    ) AS candidate ON candidate.project_id = e.project_id
    SET e.gateway_application_id = candidate.application_id,
        e.updated_at = CURRENT_TIMESTAMP
    WHERE e.gateway_application_id IS NULL;
END//

CALL _pomelo_orbit_v37_to_v38_bind_legacy_gateways()//
DROP PROCEDURE _pomelo_orbit_v37_to_v38_bind_legacy_gateways//

DELIMITER ;

DELIMITER //

CREATE PROCEDURE _pomelo_orbit_v37_to_v38_postcheck()
BEGIN
    DECLARE deployment_columns INT DEFAULT 0;
    DECLARE credential_columns INT DEFAULT 0;
    DECLARE environment_project_columns INT DEFAULT 0;
    DECLARE environment_project_unique_indexes INT DEFAULT 0;
    DECLARE environment_name_columns INT DEFAULT 0;
    DECLARE environment_foreign_keys INT DEFAULT 0;
    DECLARE projects_without_environment INT DEFAULT 0;
    DECLARE invalid_environment_credentials INT DEFAULT 0;
    DECLARE invalid_environment_gateway_bindings INT DEFAULT 0;

    SELECT COUNT(*) INTO deployment_columns
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
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
    WHERE table_schema = DATABASE()
      AND table_name = 'credential'
      AND column_name = 'revision';

    SELECT COUNT(*) INTO environment_project_columns
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'environment'
      AND column_name = 'project_id';

    SELECT COUNT(*) INTO environment_project_unique_indexes
    FROM (
        SELECT index_name
        FROM information_schema.statistics
        WHERE table_schema = DATABASE()
          AND table_name = 'environment'
          AND non_unique = 0
        GROUP BY index_name
        HAVING COUNT(*) = 1
           AND MIN(column_name) = 'project_id'
    ) AS project_unique_indexes;

    SELECT COUNT(*) INTO environment_name_columns
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'environment'
      AND column_name = 'name';

    SELECT COUNT(*) INTO environment_foreign_keys
    FROM information_schema.table_constraints
    WHERE table_schema = DATABASE()
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
       OR environment_project_unique_indexes <> 1
       OR environment_name_columns <> 0
       OR environment_foreign_keys <> 0
       OR projects_without_environment <> 0
       OR invalid_environment_credentials <> 0
       OR invalid_environment_gateway_bindings <> 0 THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'v37 to v38 schema and environment verification failed';
    END IF;
END//

CALL _pomelo_orbit_v37_to_v38_postcheck()//
DROP PROCEDURE _pomelo_orbit_v37_to_v38_postcheck//

DELIMITER ;

SELECT version, dirty FROM schema_migrations;