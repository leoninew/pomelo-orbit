import { describe, expect, it } from 'vitest';
import type { ServiceComponentDeclaredMount } from '@/gen/proto/orbit/v1/service/service';
import {
  cloneComponentMount,
  componentMountHasOverrides,
  componentMountOverlays,
  componentMountRows,
  resetComponentMount,
  setMountFieldValue,
  validateComponentMount,
} from './serviceComponentMount';

const declaration: ServiceComponentDeclaredMount = {
  source_type: 'controlled_file',
  source: './config/app.yml',
  target: '/app/config.yml',
  content: 'name: app\n',
  mode: '0600',
  ignore_if_exists: true,
  read_only: false,
  source_is_host_path: false,
  shared: false,
};

describe('service controlled file overlays', () => {
  it('saves edited fields and keeps untouched settings across a round trip', () => {
    const [row] = componentMountRows([declaration], []);
    expect(componentMountOverlays([row])).toEqual([]);
    setMountFieldValue(row, 'content', declaration.content);
    expect(componentMountHasOverrides(row)).toBe(true);
    const payload = componentMountOverlays([row]);
    expect(payload).toEqual([
      { target: declaration.target, state: 'override', content: declaration.content },
    ]);
    const [reloaded] = componentMountRows([{ ...declaration, content: 'new version' }], payload);
    expect(reloaded.values.content).toBe(declaration.content);
    resetComponentMount(reloaded);
    expect(reloaded.values.content).toBe('new version');
    expect(componentMountOverlays([reloaded])).toEqual([]);
  });

  it('preserves empty content, false policies and independent mode or source overrides', () => {
    const overlay = {
      target: declaration.target,
      state: 'override',
      content: '',
      ignore_if_exists: false,
      mode: '0000',
    };
    const [row] = componentMountRows([declaration], [overlay]);
    expect(componentMountOverlays([row])).toEqual([overlay]);
    expect(validateComponentMount(row)).toEqual({});
    setMountFieldValue(row, 'source', './config/another.yml');
    expect(componentMountOverlays([row])[0].source).toBe('./config/another.yml');
    expect(componentMountOverlays([row])[0]).toMatchObject(overlay);
    expect(componentMountOverlays([row])[0]).not.toHaveProperty('source_is_host_path');
  });

  it('isolates dialog changes and resets deleted mounts without affecting other rows', () => {
    const declarations = [declaration, { ...declaration, target: '/other.yml' }];
    const rows = componentMountRows(declarations, [
      { target: declaration.target, state: 'deleted' },
      { target: '/other.yml', state: 'override', mode: '0644' },
    ]);
    const draft = cloneComponentMount(rows[1]);
    setMountFieldValue(draft, 'content', 'draft');
    expect(rows[1].overridden.content).toBe(false);
    expect(rows[1].values.content).toBe(declaration.content);
    expect(componentMountOverlays(rows)[0]).toEqual({
      target: declaration.target,
      state: 'deleted',
    });
    resetComponentMount(rows[0]);
    expect(componentMountOverlays(rows)).toEqual([
      { target: '/other.yml', state: 'override', mode: '0644' },
    ]);
  });

  it('saves directly cleared content and unchecked preserve settings', () => {
    const [row] = componentMountRows([declaration], []);
    setMountFieldValue(row, 'content', '');
    setMountFieldValue(row, 'ignore_if_exists', false);
    expect(componentMountOverlays([row])).toEqual([
      {
        target: declaration.target,
        state: 'override',
        content: '',
        ignore_if_exists: false,
      },
    ]);
  });

  it('validates the effective path, mode and UTF-8 byte limit', () => {
    const [row] = componentMountRows([declaration], []);
    row.values.source = 'config/app.yml';
    row.values.mode = '8888';
    row.values.content = '中'.repeat(87382);
    expect(validateComponentMount(row)).toEqual({
      source: 'service.componentDetail.validation.sourceMustStartWithDotSlash',
      mode: 'service.componentDetail.validation.fileMode',
      content: 'service.componentDetail.validation.fileSize',
    });
    row.values.source = './config/app.yml';
    row.values.mode = '0644';
    row.values.content = ' '.repeat(256 * 1024);
    expect(validateComponentMount(row)).toEqual({});
    expect(row.values.content).toHaveLength(256 * 1024);
  });

  it('validates host paths and volume names without file-only checks', () => {
    const [row] = componentMountRows(
      [{ ...declaration, source_type: 'directory', mode: '', content: '' }],
      []
    );
    row.values.source_is_host_path = true;
    expect(validateComponentMount(row).source).toBeDefined();
    row.values.source = '/srv/config';
    expect(validateComponentMount(row)).toEqual({});
    setMountFieldValue(row, 'read_only', true);
    expect(componentMountOverlays([row])[0].read_only).toBe(true);
    row.source_type = 'named_volume';
    row.values.source_is_host_path = false;
    expect(validateComponentMount(row).source).toBe(
      'service.componentDetail.validation.volumeName'
    );
    row.values.source = 'data';
    expect(validateComponentMount(row)).toEqual({});
  });
});
