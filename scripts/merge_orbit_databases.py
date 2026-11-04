"""Reconcile one Orbit project between two PostgreSQL databases through dbtalk."""

from __future__ import annotations

import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
from datetime import datetime, timezone


SIDES = ('left', 'right')
EXCLUDED = {'user', 'user_role', 'role', 'role_permission', 'permission', 'login_history', 'mcp_access_token', 'schema_migrations'}
TERMINAL = {'ran_to_completion', 'faulted', 'canceled'}
TASK_TERMINAL = {'succeeded', 'failed', 'canceled'}
GLOBAL_TABLES = {'repository', 'repository_credential'}
NATURAL_KEYS = [
    ('project', ('code',)), ('environment', ('project_id',)),
    ('environment_credential', ('environment_id',)), ('application', ('project_id', 'code')),
    ('version', ('application_id', 'label')), ('service', ('project_id', 'code')),
    ('version_component', ('version_id', 'name')),
    ('service_component', ('service_id', 'component_name')),
    ('service_component_mount', ('service_component_id', 'target')),
    ('service_component_endpoint', ('service_component_id', 'protocol', 'container_port')),
    ('route', ('project_id', 'name')), ('repository', ('project_id', 'code')),
    ('repository_credential', ('project_id', 'name')),
    ('pipeline', ('project_id', 'kind', 'name')),
    ('pipeline_stage', ('pipeline_id', 'name')), ('pipeline_stage_reference', ('pipeline_id', 'name')),
]
REFERENCES = {
    'project_id': 'project', 'application_id': 'application', 'gateway_application_id': 'application',
    'environment_id': 'environment', 'ssh_credential_id': 'environment_credential',
    'service_id': 'service', 'version_id': 'version', 'source_version_id': 'version',
    'generated_version_id': 'version', 'fixed_version_id': 'version', 'created_from_version_id': 'version',
    'component_id': 'version_component', 'source_version_component_id': 'version_component',
    'service_component_id': 'service_component', 'pipeline_id': 'pipeline',
    'source_pipeline_id': 'pipeline', 'pipeline_run_id': 'pipeline_run', 'retry_of': 'pipeline_run',
    'snapshot_id': 'pipeline_snapshot', 'stage_id': 'pipeline_stage', 'pipeline_stage_id': 'pipeline_stage',
    'source_template_stage_id': 'pipeline_stage', 'repository_id': 'repository',
    'git_credential_id': 'repository_credential', 'artifact_id': 'artifact', 'source_artifact_id': 'artifact',
    'current_deployment_id': 'deployment', 'rollback_from_deployment_id': 'deployment',
    'conversation_id': 'deployment_dialogue_conversation', 'user_id': 'user', 'created_by_user_id': 'user',
}


class MergeError(Exception):
    pass


def require(condition, message):
    if not condition:
        raise MergeError(message)


def encode(value):
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(',', ':'))


def fingerprint(value):
    return hashlib.sha256(encode(value).encode('utf-8')).hexdigest()


def table_hashes(tables):
    return {table: fingerprint(sorted(rows, key=encode)) for table, rows in tables.items()}


def write_report(path, value):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    descriptor, temporary = tempfile.mkstemp(prefix=path.name + '.', dir=path.parent)
    try:
        with os.fdopen(descriptor, 'w', encoding='utf-8') as stream:
            json.dump(value, stream, ensure_ascii=False, indent=2)
            stream.write('\n')
        Path(temporary).replace(path)
    finally:
        Path(temporary).unlink(missing_ok=True)


def quote(value):
    return "'" + value.replace("'", "''") + "'"


def ident(value):
    require(re.fullmatch(r'[a-z_][a-z_0-9]*', value), 'Unsupported SQL identifier')
    return '"' + value + '"'


def timestamp(value):
    if not value:
        return datetime.min.replace(tzinfo=timezone.utc)
    parsed = datetime.fromisoformat(value)
    return parsed if parsed.tzinfo else parsed.replace(tzinfo=timezone.utc)


def latest(row):
    return timestamp(row.get('updated_at') or row.get('finished_at') or row.get('started_at') or row.get('created_at'))


def choose(entries, table, key):
    ordered = sorted(entries, key=lambda entry: latest(entry[1]), reverse=True)
    side, winner = ordered[0]
    for other_side, row in ordered[1:]:
        if latest(row) == latest(winner) and row != winner:
            fields = sorted(field for field in row if row[field] != winner.get(field))
            raise MergeError(f'Equal-time conflict: {table} key={encode(key)} fields={fields}; no source chosen')
    return side, copy.deepcopy(winner)


class Database:
    def __init__(self, side, dsn_env, cwd):
        require(re.fullmatch(r'DBTALK_DSN_[A-Z0-9_]+', dsn_env), 'Use an existing DBTALK_DSN_* variable name')
        self.side, self.dsn_env, self.cwd = side, dsn_env, Path(cwd).resolve()

    def call(self, operation, arguments):
        command = ['dbtalk', operation, '--dsn-env', self.dsn_env, '--connect-timeout', '12', '--timeout', '120', *arguments]
        try:
            result = subprocess.run(command, cwd=self.cwd, capture_output=True, encoding='utf-8', errors='replace', timeout=150)
        except (OSError, subprocess.TimeoutExpired):
            raise MergeError(f'{self.side}: dbtalk unavailable or timed out') from None
        if result.returncode:
            safe = re.findall(r'(?:Concurrent modification|Outside reference|Outside change|Result mismatch|Primary key collision): [a-z_]+', result.stdout + result.stderr)
            raise MergeError(f'{self.side}: dbtalk {operation} failed; {safe}; raw output suppressed')
        return result.stdout

    def query(self, sql):
        return json.loads(self.call('query', ['--sql', sql.replace('%', '%%'), '--format', 'json']))['rows']

    def execute(self, sql):
        descriptor, temporary = tempfile.mkstemp(prefix='orbit-merge-', suffix='.sql')
        try:
            with os.fdopen(descriptor, 'w', encoding='utf-8') as stream:
                stream.write(sql.replace('%', '%%'))
            self.call('exec', ['--file', temporary])
        finally:
            Path(temporary).unlink(missing_ok=True)

    def metadata(self):
        columns = self.query("SELECT table_name,column_name,data_type,udt_name,is_nullable,column_default,is_generated,is_identity FROM information_schema.columns WHERE table_schema='public' ORDER BY table_name,ordinal_position")
        constraints = self.query("SELECT source.relname AS table_name,constraint_record.contype AS kind,pg_get_constraintdef(constraint_record.oid) AS definition,target.relname AS target,ARRAY(SELECT attribute.attname FROM unnest(constraint_record.conkey) WITH ORDINALITY key_column(attnum,ordinality) JOIN pg_attribute attribute ON attribute.attrelid=constraint_record.conrelid AND attribute.attnum=key_column.attnum ORDER BY key_column.ordinality) AS columns FROM pg_constraint constraint_record JOIN pg_class source ON source.oid=constraint_record.conrelid JOIN pg_namespace namespace ON namespace.oid=source.relnamespace LEFT JOIN pg_class target ON target.oid=constraint_record.confrelid WHERE namespace.nspname='public' ORDER BY source.relname,constraint_record.contype,pg_get_constraintdef(constraint_record.oid)")
        indexes = self.query("SELECT tablename,indexdef FROM pg_indexes WHERE schemaname='public' ORDER BY tablename,indexdef")
        indexes = normalize_indexes(indexes)
        triggers = self.query("SELECT source.relname AS table_name FROM pg_trigger trigger_record JOIN pg_class source ON source.oid=trigger_record.tgrelid JOIN pg_namespace namespace ON namespace.oid=source.relnamespace WHERE namespace.nspname='public' AND NOT trigger_record.tgisinternal")
        require(not triggers, 'Custom database triggers require a separate review')
        return {'columns': columns, 'constraints': constraints, 'indexes': indexes}

    def snapshot(self, conditions):
        digest = "md5(coalesce(jsonb_agg(to_jsonb(entry) ORDER BY to_jsonb(entry)::text),'[]'::jsonb)::text)"
        rows = self.query(' UNION ALL '.join(f"SELECT '{table}' AS table_name,coalesce(jsonb_agg(to_jsonb(entry) ORDER BY to_jsonb(entry)::text),'[]'::jsonb) AS rows,{digest} AS digest FROM public.{ident(table)} entry WHERE {condition}" for table, condition in sorted(conditions.items())))
        return {row['table_name']: row['rows'] for row in rows}, {row['table_name']: row['digest'] for row in rows}


def normalize_indexes(indexes):
    return sorted(({'tablename': row['tablename'], 'indexdef': re.sub(r'^(CREATE (?:UNIQUE )?INDEX) (?:"(?:[^"]|"")*"|\S+) ON ', r'\1 ON ', row['indexdef'])} for row in indexes), key=encode)


def primary_keys(metadata):
    keys = {row['table_name']: row['columns'] for row in metadata['constraints'] if row['kind'] == 'p'}
    tables = {row['table_name'] for row in metadata['columns']} - EXCLUDED
    require(tables <= keys.keys(), 'Every business table needs a primary key')
    require(all(row['is_generated'] == 'NEVER' and row['is_identity'] == 'NO' for row in metadata['columns'] if row['table_name'] in tables), 'Generated/identity columns are unsupported')
    return keys


def select_project(projects, selector):
    matches = [(side, row) for side in SIDES for row in projects[side] if row['id'] == selector or row['code'] == selector]
    require(matches, 'Project selector did not match either database')
    ids = {row['id'] for side, row in matches}
    codes = {row['code'] for side, row in matches}
    while True:
        expanded = [(side, row) for side in SIDES for row in projects[side] if row['id'] in ids or row['code'] in codes]
        expanded_ids = {row['id'] for side, row in expanded}
        expanded_codes = {row['code'] for side, row in expanded}
        if expanded_ids == ids and expanded_codes == codes:
            break
        ids, codes = expanded_ids, expanded_codes
    require(all(sum(side == candidate for side, row in expanded) <= 1 for candidate in SIDES), 'Ambiguous project identity; reconcile project codes first')
    return ids


def scopes(metadata, project_ids):
    tables = {}
    for row in metadata['columns']:
        if row['table_name'] not in EXCLUDED:
            tables.setdefault(row['table_name'], set()).add(row['column_name'])
    project = 'SELECT id FROM public.project WHERE id IN (' + ','.join(quote(value) for value in sorted(project_ids)) + ')'
    apps = f'SELECT id FROM public.application WHERE project_id IN ({project})'
    versions = f'SELECT id FROM public.version WHERE application_id IN ({apps})'
    components = f'SELECT id FROM public.version_component WHERE version_id IN ({versions})'
    services = f'SELECT id FROM public.service WHERE project_id IN ({project})'
    service_components = f'SELECT id FROM public.service_component WHERE service_id IN ({services})'
    runs = f'SELECT id FROM public.pipeline_run WHERE project_id IN ({project})'
    deployments = f'SELECT id FROM public.deployment WHERE project_id IN ({project})'
    conditions = {'project': f'id IN ({project})'}
    for table, columns in tables.items():
        if table == 'project':
            continue
        if 'project_id' in columns:
            conditions[table] = f'project_id IN ({project})'
        elif table in ('version', 'gateway_config', 'gateway_acme_profile_version'):
            conditions[table] = f'application_id IN ({apps})'
        elif table == 'version_component':
            conditions[table] = f'version_id IN ({versions})'
        elif table.startswith('version_component_'):
            conditions[table] = f'component_id IN ({components})'
        elif table in ('service_component', 'service_env'):
            conditions[table] = f'service_id IN ({services})'
        elif table.startswith('service_component_'):
            conditions[table] = f'service_component_id IN ({service_components})'
        elif table == 'environment_credential':
            conditions[table] = f'environment_id IN (SELECT id FROM public.environment WHERE project_id IN ({project}))'
        elif table == 'pipeline_stage_reference':
            conditions[table] = f'pipeline_id IN (SELECT id FROM public.pipeline WHERE project_id IN ({project}))'
        elif table in ('pipeline_stage_run', 'pipeline_run_version_binding'):
            conditions[table] = f'pipeline_run_id IN ({runs})'
        elif table == 'deployment_dialogue_message':
            conditions[table] = f'conversation_id IN (SELECT id FROM public.deployment_dialogue_conversation WHERE project_id IN ({project}))'
        elif table == 'background_task':
            conditions[table] = f"payload_json::jsonb->>'project_id' IN ({project}) OR payload_json::jsonb->>'deployment_id' IN ({deployments}) OR payload_json::jsonb->>'pipeline_run_id' IN ({runs})"
        else:
            raise MergeError(f'Unclassified business table: {table}')
    return conditions


def expand_dependencies(databases, data, conditions):
    repository_ids = {row['repository_id'] for tables in data.values() for table in ('pipeline', 'pipeline_snapshot', 'pipeline_run', 'artifact') for row in tables[table] if row.get('repository_id')}
    if not repository_ids:
        return
    values = ','.join(quote(value) for value in sorted(repository_ids))
    conditions['repository'] = f"({conditions['repository']}) OR id IN ({values})"
    credential_ids = {row['git_credential_id'] for database in databases.values() for row in database.query(f'SELECT git_credential_id FROM public.repository WHERE id IN ({values})') if row['git_credential_id']}
    if credential_ids:
        values = ','.join(quote(value) for value in sorted(credential_ids))
        conditions['repository_credential'] = f"({conditions['repository_credential']}) OR id IN ({values})"


def remap(value, mapping):
    if isinstance(value, dict):
        return {key: remap(item, mapping) for key, item in value.items()}
    if isinstance(value, list):
        return [remap(item, mapping) for item in value]
    if isinstance(value, str):
        if value in mapping:
            return mapping[value]
        if value.startswith(('{', '[')):
            try:
                parsed = json.loads(value)
            except json.JSONDecodeError:
                return value
            rewritten = remap(parsed, mapping)
            if rewritten != parsed:
                return encode(rewritten)
    return value


def idle_deployments(data, deployment_ids):
    accepted = {}
    for deployment_id in deployment_ids:
        records = [[row for row in data[side]['deployment'] if row['id'] == deployment_id] for side in SIDES]
        tasks = [[row for row in data[side]['background_task'] if json.loads(row['payload_json']).get('deployment_id') == deployment_id] for side in SIDES]
        require(all(len(rows) == 1 for rows in records + tasks), f'Idle deployment needs one matching task on both sides: {deployment_id}')
        require(records[0] == records[1] and tasks[0] == tasks[1], f'Idle deployment/task differs: {deployment_id}')
        row, task = records[0][0], tasks[0][0]
        require(row['status'] == 'running' and not row['finished_at'] and task['status'] == 'succeeded' and task['finished_at'] and not task['locked_at'] and not task['locked_by'] and (datetime.now(timezone.utc) - latest(task)).days >= 7, f'Idle deployment evidence insufficient: {deployment_id}')
        accepted[deployment_id] = row
    return accepted


def merge_project(data, keys, allow_idle=()):
    working = copy.deepcopy(data)
    idle = idle_deployments(working, allow_idle)
    for tables in working.values():
        for table in ('deployment', 'pipeline_run'):
            require(all(row['status'] in TERMINAL or (table == 'deployment' and row == idle.get(row['id'])) for row in tables[table]), f'Active execution in {table}')
        require(all(row['status'] in TASK_TERMINAL for row in tables['background_task']), 'Active background task')
    project_side, project = choose([(side, row) for side in SIDES for row in working[side]['project']], 'project', 'identity')
    mapping = {}
    for table, fields in NATURAL_KEYS:
        groups = []
        for side in SIDES:
            for original in working[side][table]:
                row = remap(original, mapping)
                if table == 'environment':
                    row['code'] = project['code']
                matches = [group for group in groups if any(entry['id'] == row['id'] or tuple(entry[field] for field in fields) == tuple(row[field] for field in fields) for origin, entry in group)]
                group = [(side, row)]
                for matching in matches:
                    group.extend(matching)
                    groups.remove(matching)
                groups.append(group)
        for group in groups:
            require(all(sum(origin == side for origin, row in group) <= 1 for side in SIDES), f'Ambiguous same-side natural identity: {table}')
            newest = max(latest(entry[1]) for entry in group)
            canonical = min(entry[1]['id'] for entry in group if latest(entry[1]) == newest)
            for side, row in group:
                if row['id'] != canonical:
                    mapping[row['id']] = canonical
    for source in mapping:
        seen = {source}
        while mapping[source] in mapping:
            require(mapping[source] not in seen, 'Identity mapping cycle')
            seen.add(mapping[source])
            mapping[source] = mapping[mapping[source]]
    normalized = {side: remap(tables, mapping) for side, tables in working.items()}
    for tables in normalized.values():
        for environment in tables['environment']:
            environment['code'] = project['code']
    owners = {}
    for table in ('version', 'service'):
        groups = {}
        for side in SIDES:
            for row in normalized[side][table]:
                groups.setdefault(row['id'], []).append((side, row))
        owners[table] = {key: choose(entries, table, key)[0] for key, entries in groups.items()}
        for key, entries in groups.items():
            if len(entries) == 2 and entries[0][1] == entries[1][1]:
                require(aggregate(normalized['left'], table, key) == aggregate(normalized['right'], table, key), f'Equal parent with differing child configuration: {table} {key}')
    filtered = copy.deepcopy(normalized)
    for side, tables in filtered.items():
        for parent, component, foreign, prefix in (('version', 'version_component', 'version_id', 'component_id'), ('service', 'service_component', 'service_id', 'service_component_id')):
            tables[component] = [row for row in tables[component] if owners[parent][row[foreign]] == side]
            selected_ids = {row['id'] for row in tables[component]}
            for table in tables:
                if table.startswith(component + '_'):
                    tables[table] = [row for row in tables[table] if row[prefix] in selected_ids]
        tables['service_env'] = [row for row in tables['service_env'] if owners['service'][row['service_id']] == side]
    merged, decisions = {}, []
    for table in filtered['left']:
        groups = {}
        for side in SIDES:
            for row in filtered[side][table]:
                groups.setdefault(tuple(row[field] for field in keys[table]), []).append((side, row))
        merged[table] = []
        for key, entries in groups.items():
            side, winner = choose(entries, table, key)
            if table == 'environment':
                target_fields = ('target_type', 'platform', 'host', 'port', 'username', 'workspace_root', 'host_key_fingerprint', 'target_revision', 'last_probe_revision')
                probes = [(origin, row) for origin, row in entries if row.get('last_probe_at') and all(row[field] == winner[field] for field in target_fields)]
                if probes:
                    probe_fields = ('last_probe_revision', 'last_probe_status', 'last_probe_at', 'last_probe_diagnostic')
                    probe_entries = [(origin, {**{field: row[field] for field in probe_fields}, 'updated_at': row['last_probe_at']}) for origin, row in probes]
                    probe_side, probe = choose(probe_entries, 'environment_probe', key)
                    for field in probe_fields:
                        winner[field] = probe[field]
                    if probe_side != side:
                        decisions.append({'table': table, 'key': key, 'probe_winner': probe_side, 'rule': 'latest_probe_for_same_target_and_probe_revision'})
            merged[table].append(winner)
            if len(entries) == 2 and entries[0][1] != entries[1][1]:
                decisions.append({'table': table, 'key': key, 'winner': side, 'fields': sorted(field for field in winner if entries[0][1][field] != entries[1][1][field])})
    for deployment_id, original in idle.items():
        require([row for row in merged['deployment'] if row['id'] == deployment_id] == [original], 'Idle deployment would be modified')
    return merged, mapping, decisions


def aggregate(tables, parent, parent_id):
    component = parent + '_component'
    rows = [row for row in tables[component] if row[parent + '_id'] == parent_id]
    ids = {row['id'] for row in rows}
    foreign = 'component_id' if parent == 'version' else 'service_component_id'
    result = {component: sorted(rows, key=encode)}
    for table in tables:
        if table.startswith(component + '_'):
            result[table] = sorted((row for row in tables[table] if row[foreign] in ids), key=encode)
    if parent == 'service':
        result['service_env'] = sorted((row for row in tables['service_env'] if row['service_id'] == parent_id), key=encode)
    return result


def audit(databases, before, merged, mapping, conditions):
    removed = {row['id'] for tables in before.values() for rows in tables.values() for row in rows if 'id' in row} - {row['id'] for rows in merged.values() for row in rows if 'id' in row}
    require(not any(value in encode(merged) for value in removed), 'Merged data still contains a removed ID, including JSON/text references')
    inherited, catalogs = set(), {}
    for side, database in databases.items():
        rows = database.query(' UNION ALL '.join(f"SELECT '{table}' AS table_name,id FROM public.{ident(table)}" for table in sorted(set(REFERENCES.values()))))
        catalogs[side] = {table: {row['id'] for row in rows if row['table_name'] == table} for table in set(REFERENCES.values())}
        for table, entries in before[side].items():
            for row in entries:
                for field, target in REFERENCES.items():
                    if row.get(field) and row[field] not in catalogs[side][target]:
                        inherited.add((table, field, mapping.get(row[field], row[field])))
    missing = set()
    for side, database in databases.items():
        known = {table: (ids - removed) | {row.get('id') for row in merged.get(table, [])} for table, ids in catalogs[side].items()}
        for table, entries in merged.items():
            for row in entries:
                for field, target in REFERENCES.items():
                    if row.get(field) and row[field] not in known[target]:
                        missing.add((table, field, row[field]))
        if removed:
            pattern = quote('|'.join(sorted(removed)))
            outside = database.query(' UNION ALL '.join(f"SELECT '{table}' AS table_name,count(*) AS count FROM public.{ident(table)} entry WHERE NOT coalesce(({condition}),false) AND to_jsonb(entry)::text ~ {pattern} HAVING count(*)>0" for table, condition in sorted(conditions.items())))
            require(not outside, f'{side}: references to removed IDs outside selected project')
    require(not missing - inherited, 'Merge would introduce missing business references')
    semantic_audit(merged)
    return sorted(missing), removed


def semantic_audit(tables):
    indexes = {table: {row['id']: row for row in tables[table]} for table in ('application', 'version', 'service', 'deployment')}
    for service in tables['service']:
        version = indexes['version'].get(service['version_id'])
        app = indexes['application'].get(service['application_id'])
        deployment = indexes['deployment'].get(service.get('current_deployment_id'))
        require(not version or version['application_id'] == service['application_id'], 'Service/version application mismatch')
        require(not app or app['project_id'] == service['project_id'], 'Service/application project mismatch')
        require(not deployment or (deployment['service_id'] == service['id'] and deployment['project_id'] == service['project_id']), 'Current deployment/service mismatch')


def check_credentials(data, merged, key_names):
    encrypted = {row[field] for tables in [*data.values(), merged] for table, field in (('environment_credential', 'encrypted_private_key'), ('repository_credential', 'encrypted_data')) for row in tables[table] if row[field]}
    if not encrypted:
        return
    require(all(key_names.values()), 'Encrypted credentials require --left-key-env and --right-key-env (process variables)')
    try:
        from cryptography.fernet import Fernet
        for side in SIDES:
            codec = Fernet(os.environ[key_names[side]])
            for value in encrypted:
                codec.decrypt(value.encode())
    except (ImportError, KeyError):
        raise MergeError('Install cryptography and set the named process key variables') from None
    except Exception:
        raise MergeError('Credentials cannot be decrypted with both instance keys; key migration is outside this tool') from None


def table_order(metadata, conditions):
    pending, ordered = set(conditions), []
    edges = [row for row in metadata['constraints'] if row['kind'] == 'f']
    while pending:
        ready = sorted(table for table in pending if not any(row['table_name'] == table and row['target'] in pending and row['target'] != table for row in edges))
        require(ready, 'Cyclic foreign keys need a separate migration')
        ordered.extend(ready)
        pending.difference_update(ready)
    return ordered


def transaction_sql(before, merged, conditions, keys, order, hashes, removed, rehearsal):
    payload = encode(merged)
    require('$orbit_merge$' not in payload, 'Unsupported SQL delimiter in data')
    digest = "md5(coalesce(jsonb_agg(to_jsonb(entry) ORDER BY to_jsonb(entry)::text),'[]'::jsonb)::text)"
    statements = ["PERFORM set_config('search_path','public,pg_temp',true);", "PERFORM set_config('lock_timeout','10s',true);", "PERFORM set_config('statement_timeout','120s',true);", 'LOCK TABLE ' + ','.join('public.' + ident(table) for table in sorted(conditions)) + ' IN SHARE ROW EXCLUSIVE MODE;', 'CREATE TEMP TABLE orbit_merge_outside(table_name text PRIMARY KEY,digest text) ON COMMIT DROP;']
    for table, condition in sorted(conditions.items()):
        statements.append(f"IF (SELECT {digest} FROM {ident(table)} entry WHERE {condition}) <> {quote(hashes[table])} THEN RAISE EXCEPTION 'Concurrent modification: {table}'; END IF;")
        statements.append(f"INSERT INTO orbit_merge_outside SELECT '{table}',{digest} FROM {ident(table)} entry WHERE NOT coalesce(({condition}),false);")
        if removed:
            pattern = quote('|'.join(sorted(removed)))
            statements.append(f"IF EXISTS (SELECT 1 FROM {ident(table)} entry WHERE NOT coalesce(({condition}),false) AND to_jsonb(entry)::text ~ {pattern}) THEN RAISE EXCEPTION 'Outside reference: {table}'; END IF;")
        if merged[table]:
            matches = ' AND '.join(f'original.{ident(field)}=incoming.{ident(field)}' for field in keys[table])
            original_keys = ','.join(f'original.{ident(field)}' for field in keys[table])
            selected_keys = ','.join(ident(field) for field in keys[table])
            statements.append(f"IF EXISTS (SELECT 1 FROM {ident(table)} original JOIN jsonb_populate_recordset(NULL::{ident(table)},payload->'{table}') incoming ON {matches} WHERE ({original_keys}) NOT IN (SELECT {selected_keys} FROM {ident(table)} WHERE {condition})) THEN RAISE EXCEPTION 'Primary key collision: {table}'; END IF;")
    for table in reversed(order):
        rows = [row for row in before[table] if not (table in GLOBAL_TABLES and row['project_id'] is None)]
        if rows:
            values = encode([{field: row[field] for field in keys[table]} for row in rows])
            matches = ' AND '.join(f'existing.{ident(field)}=original.{ident(field)}' for field in keys[table])
            statements.append(f'DELETE FROM {ident(table)} existing USING jsonb_populate_recordset(NULL::{ident(table)},{quote(values)}::jsonb) original WHERE {matches};')
    for table in order:
        if merged[table]:
            columns = ','.join(ident(field) for field in keys[table])
            updates = ','.join(f'{ident(field)}=excluded.{ident(field)}' for field in merged[table][0] if field not in keys[table])
            conflict = 'DO UPDATE SET ' + updates if updates else 'DO NOTHING'
            statements.append(f"INSERT INTO {ident(table)} SELECT * FROM jsonb_populate_recordset(NULL::{ident(table)},payload->'{table}') WHERE true ON CONFLICT ({columns}) {conflict};")
    for table, condition in sorted(conditions.items()):
        statements.append(f"IF (SELECT {digest} FROM {ident(table)} entry WHERE NOT coalesce(({condition}),false)) <> (SELECT digest FROM orbit_merge_outside WHERE table_name='{table}') THEN RAISE EXCEPTION 'Outside change: {table}'; END IF;")
        statements.append(f"IF (SELECT coalesce(jsonb_agg(to_jsonb(entry) ORDER BY to_jsonb(entry)::text),'[]'::jsonb) FROM {ident(table)} entry WHERE {condition}) <> (SELECT coalesce(jsonb_agg(value ORDER BY value::text),'[]'::jsonb) FROM jsonb_array_elements(payload->'{table}')) THEN RAISE EXCEPTION 'Result mismatch: {table}'; END IF;")
    body = '\n'.join(statements)
    if rehearsal:
        body = "BEGIN\n" + body + "\nRAISE EXCEPTION 'ORBIT_MERGE_ROLLBACK';\nEXCEPTION WHEN raise_exception THEN IF SQLERRM <> 'ORBIT_MERGE_ROLLBACK' THEN RAISE; END IF;\nEND;"
    return 'DO $orbit_merge$ DECLARE payload jsonb := ' + quote(payload) + '::jsonb; BEGIN\n' + body + '\nEND $orbit_merge$;'


def load_scope(databases, metadata, selector, saved_ids=None, dependency_ids=None):
    if saved_ids is None:
        projects = {side: database.query('SELECT id,code FROM public.project') for side, database in databases.items()}
        saved_ids = sorted(select_project(projects, selector))
    conditions = scopes(metadata, saved_ids)
    data = {side: database.snapshot(conditions)[0] for side, database in databases.items()}
    if dependency_ids is None:
        expand_dependencies(databases, data, conditions)
    else:
        for table in GLOBAL_TABLES:
            if dependency_ids[table]:
                values = ','.join(quote(value) for value in dependency_ids[table])
                conditions[table] = f'({conditions[table]}) OR id IN ({values})'
    snapshots = {side: database.snapshot(conditions) for side, database in databases.items()}
    return saved_ids, conditions, {side: value[0] for side, value in snapshots.items()}, {side: value[1] for side, value in snapshots.items()}


def options_parser():
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest='command', required=True)
    for command in ('plan', 'apply', 'verify'):
        subparser = subparsers.add_parser(command)
        subparser.add_argument('--left-dsn-env', required=True)
        subparser.add_argument('--right-dsn-env', required=True)
        subparser.add_argument('--cwd', type=Path, default=Path.cwd(), help='Directory containing dbtalk .env')
        if command == 'apply':
            subparser.add_argument('--plan', type=Path, required=True)
        else:
            subparser.add_argument('--project', required=True, help='Exact project code or ID on either side')
            subparser.add_argument('--output', type=Path, required=True, help='Local JSON report with complete business rows and differences')
        if command in ('plan', 'apply'):
            subparser.add_argument('--left-key-env')
            subparser.add_argument('--right-key-env')
        if command == 'plan':
            subparser.add_argument('--allow-idle-deployment', action='append', default=[], metavar='ID')
    return parser


def run_command(args):
    require(args.left_dsn_env != args.right_dsn_env, 'Choose two different DSN variable names')
    databases = {side: Database(side, getattr(args, side + '_dsn_env'), args.cwd) for side in SIDES}
    identities = {side: database.query('SELECT current_database() AS database_name,inet_server_addr()::text AS server_address,inet_server_port() AS server_port,pg_postmaster_start_time() AS server_started_at')[0] for side, database in databases.items()}
    require(identities['left'] != identities['right'], 'Both DSN variables resolve to the same database')
    metadata = {side: database.metadata() for side, database in databases.items()}
    require(metadata['left'] == metadata['right'], 'Database schemas differ')
    metadata = metadata['left']
    keys = primary_keys(metadata)
    schema_hash = fingerprint(metadata)
    endpoints = {side: database.dsn_env for side, database in databases.items()}
    key_names = {side: getattr(args, side + '_key_env', None) for side in SIDES}
    saved = None
    if args.command == 'apply':
        saved = json.loads(args.plan.read_text(encoding='utf-8'))
        require(saved.get('format') == 1 and saved.get('status') == 'ready', 'A successful plan is required')
        require(saved['endpoints'] == endpoints and saved['cwd'] == str(args.cwd.resolve()), 'Plan endpoints or dbtalk directory changed')
        require(saved['database_identities'] == identities, 'DSN variables now resolve to different database endpoints')
        require(saved['schema_hash'] == schema_hash, 'Schema changed since plan')
    selector = saved['project'] if saved else args.project
    ids, conditions, data, hashes = load_scope(databases, metadata, selector, saved['project_ids'] if saved else None, saved['dependency_ids'] if saved else None)
    report = {'format': 1, 'observed_at': datetime.now(timezone.utc).isoformat(), 'project': selector, 'project_ids': ids, 'endpoints': endpoints, 'database_identities': identities, 'cwd': str(args.cwd.resolve()), 'schema_hash': schema_hash, 'before': {side: table_hashes(tables) for side, tables in data.items()}, 'before_rows': data, 'differences': row_differences(data['left'], data['right'], keys), 'counts_before': {side: {table: len(rows) for table, rows in tables.items()} for side, tables in data.items()}}
    if args.command == 'verify':
        report['identical'] = report['before']['left'] == report['before']['right']
        report['different_tables'] = [table for table in data['left'] if report['before']['left'][table] != report['before']['right'][table]]
        write_report(args.output, report)
        return 0 if report['identical'] else 2
    if saved:
        require(all(table_hashes(saved['before_rows'][side]) == saved['before'][side] for side in SIDES) and table_hashes(saved['merged_rows']) == saved['expected'], 'Plan rows do not match plan fingerprints')
        states = {side: classify_state(report['before'][side], saved['before'][side], saved['expected']) for side in SIDES}
        committed = [side for side in SIDES if states[side] == 'expected']
        if committed:
            merged, mapping, decisions = copy.deepcopy(data[committed[0]]), saved['id_mapping'], saved['decisions']
        else:
            merged, mapping, decisions = merge_project(data, keys, saved['allow_idle_deployments'])
        require(table_hashes(merged) == saved['expected'], 'Recomputed merge differs from reviewed plan')
    else:
        report.update({'allow_idle_deployments': args.allow_idle_deployment})
        report.update({'status': 'blocked', 'blocker': 'Plan validation has not completed'})
        write_report(args.output, report)
        try:
            merged, mapping, decisions = merge_project(data, keys, args.allow_idle_deployment)
        except MergeError as error:
            report.update({'status': 'blocked', 'blocker': str(error)})
            write_report(args.output, report)
            raise
    try:
        inherited, removed = audit(databases, data, merged, mapping, conditions)
        check_credentials(data, merged, key_names)
    except MergeError as error:
        if not saved:
            report.update({'status': 'blocked', 'blocker': str(error)})
            write_report(args.output, report)
        raise
    if not saved:
        report.pop('blocker', None)
        report.update({'status': 'ready', 'expected': table_hashes(merged), 'merged_rows': merged, 'changes_by_side': {side: row_differences(data[side], merged, keys) for side in SIDES}, 'dependency_ids': {table: sorted({row['id'] for tables in data.values() for row in tables[table]}) for table in GLOBAL_TABLES}, 'counts_after': {table: len(rows) for table, rows in merged.items()}, 'id_mapping': mapping, 'decisions': decisions, 'inherited_missing_references': inherited, 'environment_after': merged['environment']})
        write_report(args.output, report)
        return 0
    order = table_order(metadata, conditions)
    pending = [side for side in SIDES if states[side] != 'expected']
    journal_path = args.plan.with_suffix('.execution.json')
    journal = {'plan': str(args.plan.resolve()), 'plan_hash': fingerprint(saved), 'started_at': datetime.now(timezone.utc).isoformat(), 'sides': {side: states[side] for side in SIDES}, 'actual_rows': {}, 'verification_differences': {}}
    write_report(journal_path, journal)
    for side in pending:
        databases[side].execute(transaction_sql(data[side], merged, conditions, keys, order, hashes[side], removed, True))
        current, current_hashes = databases[side].snapshot(conditions)
        require(current_hashes == hashes[side], f'{side}: rehearsal did not leave the original snapshot unchanged')
        journal['sides'][side] = 'rehearsed'
        write_report(journal_path, journal)
    for side in pending:
        databases[side].execute(transaction_sql(data[side], merged, conditions, keys, order, hashes[side], removed, False))
        journal['sides'][side] = 'committed'
        write_report(journal_path, journal)
    for side in SIDES:
        actual, current_hashes = databases[side].snapshot(conditions)
        journal['actual_rows'][side] = actual
        journal['verification_differences'][side] = row_differences(merged, actual, keys)
        write_report(journal_path, journal)
        require(table_hashes(actual) == saved['expected'], f'{side}: post-commit verification differs; inspect execution journal')
        journal['sides'][side] = 'verified'
        write_report(journal_path, journal)
    return 0


def classify_state(current, before, expected):
    if current == expected:
        return 'expected'
    require(current == before, 'Data changed since plan; generate a new plan instead of overwriting it')
    return 'before'


def row_differences(left, right, keys):
    changes = []
    for table in sorted(left.keys() | right.keys()):
        sources = [{tuple(row[field] for field in keys[table]): row for row in tables.get(table, [])} for tables in (left, right)]
        for key in sorted(sources[0].keys() | sources[1].keys(), key=encode):
            previous, following = sources[0].get(key), sources[1].get(key)
            if previous == following:
                continue
            fields = sorted(field for field in (previous or {}).keys() | (following or {}).keys() if previous is None or following is None or previous.get(field) != following.get(field))
            changes.append({'table': table, 'key': dict(zip(keys[table], key)), 'operation': 'insert' if previous is None else 'delete' if following is None else 'update', 'fields': fields, 'before': previous, 'after': following})
    return changes


def main(argv=None):
    args = options_parser().parse_args(argv)
    try:
        result = run_command(args)
        print(encode({'command': args.command, 'status': 'ok' if result == 0 else 'different'}))
        return result
    except MergeError as error:
        print(encode({'status': 'blocked', 'message': str(error)}), file=sys.stderr)
        return 1
    except (OSError, ValueError, KeyError, TypeError):
        print(encode({'status': 'blocked', 'message': 'Invalid input, report or database response; details suppressed to protect secrets'}), file=sys.stderr)
        return 1


if __name__ == '__main__':
    sys.exit(main())
