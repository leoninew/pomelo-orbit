import copy
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import merge_orbit_databases as merge


OLDER = '2026-01-01T00:00:00+00:00'
NEWER = '2026-02-01T00:00:00+00:00'


def fixture():
    names = {table for table, fields in merge.NATURAL_KEYS}
    names.update({'background_task', 'deployment', 'pipeline_run', 'pipeline_run_version_binding', 'pipeline_stage_run', 'artifact', 'pipeline_snapshot', 'gateway_config', 'gateway_acme_profile_version', 'service_env', 'version_component_env', 'version_component_mount', 'service_component_env'})
    tables = {name: [] for name in names}
    tables['project'] = [{'id': 'project', 'code': 'demo', 'updated_at': OLDER}]
    keys = {name: ['id'] for name in names}
    keys['service_env'] = ['service_id', 'env_key']
    return {'left': copy.deepcopy(tables), 'right': copy.deepcopy(tables)}, keys


def environment(updated, probe):
    return {'id': 'environment', 'project_id': 'project', 'code': 'demo', 'target_type': 'ssh', 'platform': 'linux', 'host': 'host.invalid', 'port': 22, 'username': 'operator', 'workspace_root': '/orbit', 'host_key_fingerprint': None, 'target_revision': 5, 'last_probe_revision': 4, 'last_probe_status': 'succeeded', 'last_probe_at': probe, 'last_probe_diagnostic': None, 'updated_at': updated}


class MergeTests(unittest.TestCase):
    def test_same_id_rename_is_one_project_and_selector_must_exist(self):
        projects = {'left': [{'id': 'project', 'code': 'old'}], 'right': [{'id': 'project', 'code': 'new'}]}
        self.assertEqual(merge.select_project(projects, 'new'), {'project'})
        with self.assertRaises(merge.MergeError):
            merge.select_project(projects, 'missing')
        projects['left'].append({'id': 'other', 'code': 'new'})
        with self.assertRaises(merge.MergeError):
            merge.select_project(projects, 'new')

    def test_newest_identity_rewrites_children_and_json_and_unions_versions(self):
        data, keys = fixture()
        for side, app_id, updated in (('left', 'app-old', OLDER), ('right', 'app-new', NEWER)):
            data[side]['application'] = [{'id': app_id, 'project_id': 'project', 'code': 'app', 'updated_at': updated}]
            data[side]['version'] = [{'id': side + '-version', 'application_id': app_id, 'label': side, 'updated_at': updated}]
            data[side]['background_task'] = [{'id': side + '-task', 'status': 'succeeded', 'payload_json': json.dumps({'application_id': app_id, 'nested': [app_id]}), 'updated_at': updated}]
        merged, mapping, decisions = merge.merge_project(data, keys)
        self.assertEqual(mapping['app-old'], 'app-new')
        self.assertEqual(len(merged['application']), 1)
        self.assertEqual(len(merged['version']), 2)
        self.assertTrue(all(row['application_id'] == 'app-new' for row in merged['version']))
        self.assertTrue(all(json.loads(row['payload_json'])['nested'] == ['app-new'] for row in merged['background_task']))

    def test_newer_parent_owns_complete_configuration_and_older_only_child_is_removed(self):
        data, keys = fixture()
        for side, updated in (('left', OLDER), ('right', NEWER)):
            data[side]['version'] = [{'id': 'version', 'application_id': 'app', 'label': 'base', 'updated_at': updated}]
            data[side]['version_component'] = [{'id': side + '-component', 'version_id': 'version', 'name': side, 'updated_at': updated}]
        merged, mapping, decisions = merge.merge_project(data, keys)
        self.assertEqual([row['name'] for row in merged['version_component']], ['right'])

    def test_equal_parent_cannot_silently_discard_different_children(self):
        data, keys = fixture()
        for side in merge.SIDES:
            data[side]['service'] = [{'id': 'service', 'project_id': 'project', 'code': 'web', 'updated_at': OLDER}]
            data[side]['service_env'] = [{'service_id': 'service', 'env_key': 'TOKEN', 'value': side + '-secret'}]
        with self.assertRaisesRegex(merge.MergeError, 'differing child configuration') as failure:
            merge.merge_project(data, keys)
        self.assertNotIn('secret', str(failure.exception))

    def test_equal_time_conflict_stops_without_logging_values(self):
        entries = [('left', {'id': 'run', 'created_at': OLDER, 'value': 'left-secret'}), ('right', {'id': 'run', 'created_at': OLDER, 'value': 'right-secret'})]
        with self.assertRaisesRegex(merge.MergeError, 'Equal-time conflict') as failure:
            merge.choose(entries, 'pipeline_run', ['run'])
        self.assertNotIn('secret', str(failure.exception))

    def test_finished_time_selects_latest_execution_and_naive_time_is_utc(self):
        entries = [('left', {'id': 'run', 'created_at': OLDER, 'finished_at': OLDER}), ('right', {'id': 'run', 'created_at': OLDER, 'finished_at': NEWER})]
        self.assertEqual(merge.choose(entries, 'pipeline_run', ['run'])[0], 'right')
        self.assertEqual(merge.timestamp('2026-01-01T00:00:00'), merge.timestamp(OLDER))

    def test_probe_uses_own_clock_without_claiming_current_revision_was_probed(self):
        data, keys = fixture()
        data['left']['environment'] = [environment(NEWER, OLDER)]
        data['right']['environment'] = [environment(OLDER, NEWER)]
        merged, mapping, decisions = merge.merge_project(data, keys)
        self.assertEqual(merged['environment'][0]['updated_at'], NEWER)
        self.assertEqual(merged['environment'][0]['last_probe_at'], NEWER)
        self.assertEqual(merged['environment'][0]['last_probe_revision'], 4)
        self.assertEqual(merged['environment'][0]['target_revision'], 5)
        data['right']['environment'][0]['host'] = 'other.invalid'
        merged, mapping, decisions = merge.merge_project(data, keys)
        self.assertEqual(merged['environment'][0]['last_probe_at'], OLDER)

    def test_project_rename_propagates_to_environment(self):
        data, keys = fixture()
        data['right']['project'][0].update(code='renamed', updated_at=NEWER)
        for side in merge.SIDES:
            data[side]['environment'] = [environment(OLDER, OLDER)]
        merged, mapping, decisions = merge.merge_project(data, keys)
        self.assertEqual(merged['environment'][0]['code'], 'renamed')

    def test_active_execution_is_never_overwritten_by_other_side(self):
        data, keys = fixture()
        data['left']['pipeline_run'] = [{'id': 'run', 'status': 'running', 'created_at': OLDER}]
        data['right']['pipeline_run'] = [{'id': 'run', 'status': 'ran_to_completion', 'created_at': NEWER}]
        with self.assertRaisesRegex(merge.MergeError, 'Active execution'):
            merge.merge_project(data, keys)

    def test_idle_exception_requires_explicit_id_identical_rows_and_terminal_unlocked_task(self):
        data, keys = fixture()
        for side in merge.SIDES:
            data[side]['deployment'] = [{'id': 'deployment', 'status': 'running', 'finished_at': None, 'started_at': OLDER}]
            data[side]['background_task'] = [{'id': 'task', 'status': 'succeeded', 'payload_json': '{"deployment_id":"deployment"}', 'finished_at': OLDER, 'updated_at': OLDER, 'locked_at': None, 'locked_by': None}]
        with self.assertRaisesRegex(merge.MergeError, 'Active execution'):
            merge.merge_project(data, keys)
        merged, mapping, decisions = merge.merge_project(data, keys, allow_idle=['deployment'])
        self.assertEqual(merged['deployment'], data['left']['deployment'])
        data['right']['background_task'][0]['locked_by'] = 'worker'
        with self.assertRaises(merge.MergeError):
            merge.merge_project(data, keys, allow_idle=['deployment'])

    def test_distinct_pipeline_history_is_unioned_with_its_related_records(self):
        data, keys = fixture()
        for side in merge.SIDES:
            data[side]['pipeline_run'] = [{'id': side + '-run', 'status': 'ran_to_completion', 'finished_at': OLDER}]
            data[side]['artifact'] = [{'id': side + '-artifact', 'pipeline_run_id': side + '-run'}]
            data[side]['pipeline_stage_run'] = [{'id': side + '-stage', 'pipeline_run_id': side + '-run', 'status': 'ran_to_completion'}]
            data[side]['pipeline'] = [{'id': 'definition', 'project_id': 'project', 'kind': 'build', 'name': 'build', 'updated_at': OLDER}]
        merged, mapping, decisions = merge.merge_project(data, keys)
        self.assertEqual({row['id'] for row in merged['pipeline_run']}, {'left-run', 'right-run'})
        self.assertEqual({row['pipeline_run_id'] for row in merged['artifact']}, {'left-run', 'right-run'})
        self.assertEqual({row['pipeline_run_id'] for row in merged['pipeline_stage_run']}, {'left-run', 'right-run'})
        self.assertEqual(len(merged['pipeline']), 1)
        self.assertEqual(len(data['left']['pipeline_run']), 1)

    def test_snapshot_state_allows_completed_side_but_refuses_drift(self):
        self.assertEqual(merge.classify_state({'hash': 'old'}, {'hash': 'old'}, {'hash': 'new'}), 'before')
        self.assertEqual(merge.classify_state({'hash': 'new'}, {'hash': 'old'}, {'hash': 'new'}), 'expected')
        with self.assertRaisesRegex(merge.MergeError, 'Data changed'):
            merge.classify_state({'hash': 'other'}, {'hash': 'old'}, {'hash': 'new'})


class TransactionTests(unittest.TestCase):
    def test_index_names_are_not_schema_differences_but_constraints_are(self):
        left = [{'tablename': 'credential', 'indexdef': 'CREATE UNIQUE INDEX old_pkey ON public.credential USING btree (id)'}]
        right = [{'tablename': 'credential', 'indexdef': 'CREATE UNIQUE INDEX new_pkey ON public.credential USING btree (id)'}]
        self.assertEqual(merge.normalize_indexes(left), merge.normalize_indexes(right))
        right[0]['indexdef'] = right[0]['indexdef'].replace('UNIQUE ', '')
        self.assertNotEqual(merge.normalize_indexes(left), merge.normalize_indexes(right))

    def test_rehearsal_is_one_do_with_inner_exception_rollback_and_scoped_guards(self):
        before = {'project': [{'id': 'old', 'code': 'test'}]}
        after = {'project': [{'id': 'new', 'code': "text'100%"}]}
        sql = merge.transaction_sql(before, after, {'project': "id IN ('old','new')"}, {'project': ['id']}, ['project'], {'project': 'digest'}, {'old'}, True)
        self.assertTrue(sql.startswith('DO $orbit_merge$'))
        self.assertIn("RAISE EXCEPTION 'ORBIT_MERGE_ROLLBACK'", sql)
        self.assertIn('EXCEPTION WHEN raise_exception', sql)
        self.assertIn("IF SQLERRM <> 'ORBIT_MERGE_ROLLBACK' THEN RAISE", sql)
        self.assertLess(sql.index('LOCK TABLE'), sql.index('Concurrent modification'))
        self.assertLess(sql.index('Outside reference'), sql.index('DELETE FROM'))
        self.assertIn("text''100%", sql)
        self.assertNotIn('\nROLLBACK;', sql)

    def test_driver_escapes_percent_once_and_never_prints_failed_sql(self):
        database = merge.Database('left', 'DBTALK_DSN_TEST', '.')
        seen = []
        def call(operation, arguments):
            seen.append(Path(arguments[1]).read_text(encoding='utf-8'))
            raise merge.MergeError('simulated failure')
        with patch.object(database, 'call', side_effect=call):
            with self.assertRaises(merge.MergeError):
                database.execute("DO $$ BEGIN PERFORM '100%'; END $$;")
        self.assertIn('100%%', seen[0])

    def test_unknown_schema_table_fails_closed(self):
        metadata = {'columns': [{'table_name': 'new_business_table', 'column_name': 'id'}]}
        with self.assertRaisesRegex(merge.MergeError, 'Unclassified'):
            merge.scopes(metadata, {'project'})

    def test_partial_commit_resume_skips_the_committed_side(self):
        data, keys = fixture()
        data['right']['project'][0]['updated_at'] = NEWER
        data['left']['version'] = [{'id': 'left-version', 'application_id': 'app', 'label': 'left', 'updated_at': OLDER}]
        merged, mapping, decisions = merge.merge_project(data, keys)
        conditions = {table: 'false' for table in data['left']}
        conditions['project'] = "id='project'"
        metadata = {'columns': [], 'constraints': [], 'indexes': []}
        states = copy.deepcopy(data)
        commits = []
        failing = [True]

        class FakeDatabase:
            def __init__(self, side, dsn_env, cwd):
                self.side, self.dsn_env = side, dsn_env
            def metadata(self):
                return metadata
            def query(self, sql):
                return [{'database_name': self.side, 'server_address': '127.0.0.1', 'server_port': 5432}]
            def snapshot(self, selected):
                return copy.deepcopy(states[self.side]), merge.table_hashes(states[self.side])
            def execute(self, sql):
                if "RAISE EXCEPTION 'ORBIT_MERGE_ROLLBACK'" in sql:
                    return
                if self.side == 'right' and failing[0]:
                    failing[0] = False
                    raise merge.MergeError('second database disconnected')
                commits.append(self.side)
                states[self.side] = copy.deepcopy(merged)

        def load(*args, **kwargs):
            return ['project'], conditions, copy.deepcopy(states), {side: merge.table_hashes(tables) for side, tables in states.items()}

        with tempfile.TemporaryDirectory() as directory:
            plan_path = Path(directory) / 'plan.json'
            saved = {'format': 1, 'status': 'ready', 'endpoints': {'left': 'DBTALK_DSN_LEFT', 'right': 'DBTALK_DSN_RIGHT'}, 'cwd': str(Path('.').resolve()), 'project': 'demo', 'project_ids': ['project'], 'schema_hash': merge.fingerprint(metadata), 'before': {side: merge.table_hashes(tables) for side, tables in data.items()}, 'before_rows': data, 'merged_rows': merged, 'expected': merge.table_hashes(merged), 'dependency_ids': {}, 'id_mapping': mapping, 'decisions': decisions, 'allow_idle_deployments': []}
            saved['database_identities'] = {side: {'database_name': side, 'server_address': '127.0.0.1', 'server_port': 5432} for side in merge.SIDES}
            merge.write_report(plan_path, saved)
            args = merge.options_parser().parse_args(['apply', '--left-dsn-env', 'DBTALK_DSN_LEFT', '--right-dsn-env', 'DBTALK_DSN_RIGHT', '--plan', str(plan_path)])
            with patch.object(merge, 'Database', FakeDatabase), patch.object(merge, 'primary_keys', return_value=keys), patch.object(merge, 'load_scope', side_effect=load), patch.object(merge, 'audit', return_value=([], set())), patch.object(merge, 'check_credentials'):
                with self.assertRaisesRegex(merge.MergeError, 'disconnected'):
                    merge.run_command(args)
                self.assertEqual(commits, ['left'])
                self.assertEqual(merge.run_command(args), 0)
            self.assertEqual(commits, ['left', 'right'])
            journal = json.loads(plan_path.with_suffix('.execution.json').read_text(encoding='utf-8'))
            self.assertEqual(journal['sides'], {'left': 'verified', 'right': 'verified'})
            self.assertEqual(journal['actual_rows'], {'left': merged, 'right': merged})

    def test_local_report_preserves_complete_values_including_business_secrets(self):
        previous = {'service_env': [{'service_id': 'service', 'env_key': 'TOKEN', 'value': 'old-private-value'}]}
        following = {'service_env': [{'service_id': 'service', 'env_key': 'TOKEN', 'value': 'new-private-value'}]}
        changes = merge.row_differences(previous, following, {'service_env': ['service_id', 'env_key']})
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'report.json'
            merge.write_report(path, {'before_rows': previous, 'merged_rows': following, 'changes': changes})
            report = json.loads(path.read_text(encoding='utf-8'))
        self.assertEqual(report['changes'][0]['before']['value'], 'old-private-value')
        self.assertEqual(report['changes'][0]['after']['value'], 'new-private-value')
        self.assertEqual(report['before_rows'], previous)
        self.assertEqual(report['merged_rows'], following)


if __name__ == '__main__':
    unittest.main()
