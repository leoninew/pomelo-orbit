import { describe, expect, it } from 'vitest';
import {
  isFileMountRow,
  parseEnvJson,
  parsePortsJson,
  serializeEnvRows,
  serializeMountRows,
  serializePortsRows,
} from './versionComponentForm';

describe('versionComponentForm', () => {
  it('parses and serializes environment rows', () => {
    expect(parseEnvJson('[{"key":"PORT","value":8080},{"key":"","value":""}]')).toEqual([
      { key: 'PORT', value: '8080' },
    ]);
    expect(
      serializeEnvRows([
        { key: ' PORT ', value: '8080' },
        { key: '', value: 'ignored' },
      ])
    ).toBe('[{"key":"PORT","value":"8080"}]');
  });

  it('supports legacy port formats and serializes valid pairs', () => {
    expect(parsePortsJson('[80,"443","127.0.0.1:8080:80",{"published":3000,"target":3000}]')).toEqual([
      { host_port: 80, container_port: 80 },
      { host_port: 443, container_port: 443 },
      { host_port: 8080, container_port: 80 },
      { host_port: 3000, container_port: 3000 },
    ]);
    expect(
      serializePortsRows([
        { host_port: 80, container_port: 8080 },
        { host_port: 0, container_port: 80 },
      ])
    ).toBe('["80:8080"]');
  });

  it('keeps file mount content while omitting it from directory mounts', () => {
    const rows = [
      {
        source_type: 'logical',
        source: './settings.json',
        target: '/app/settings.json',
        read_only: true,
        content: '{"debug":true}',
        content_mode: 'sync',
      },
      {
        source_type: 'logical',
        source: './data',
        target: '/app/data',
        read_only: false,
        content: 'not persisted',
        content_mode: 'seed',
      },
    ];

    expect(isFileMountRow(rows[0])).toBe(true);
    expect(isFileMountRow(rows[1])).toBe(false);
    expect(serializeMountRows(rows)).toBe(
      '[{"source_type":"logical","source":"./settings.json","target":"/app/settings.json","read_only":true,"content":"{\\"debug\\":true}","content_mode":"sync"},{"source_type":"logical","source":"./data","target":"/app/data","read_only":false}]'
    );
  });
});
