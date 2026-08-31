import { describe, expect, it } from 'vitest';
import {
  componentBasicRequestFromForm,
  componentCreateRequestFromForm,
  componentDependenciesRequestFromForm,
  componentEnvRequestFromForm,
  componentMountsRequestFromForm,
  componentEndpointsRequestFromForm,
  componentRequestFromForm,
  componentAdvancedRequestFromForm,
  componentResourcesRequestFromForm,
  componentRuntimeRequestFromForm,
  componentTmpfsRequestFromForm,
  componentUlimitsRequestFromForm,
  emptyComponentForm,
} from './componentForm';

describe('componentForm', () => {
  it('preserves entered configuration text', () => {
    const form = emptyComponentForm();
    form.name = 'api';
    form.image = 'nginx:1.27';
    form.entrypoint = '  /docker-entrypoint.sh  ';
    form.command = '  nginx  ';
    form.env.push({ key: ' TOKEN ', value: ' ${TOKEN} ' });
    form.mounts.push({
      source_type: 'controlled_file',
      source: './config/app.conf',
      target: '/etc/app.conf',
      read_only: true,
      source_is_host_path: false,
      content: ' key = value ',
      mode: '0644',
      ignore_if_exists: false,
    });

    const result = componentRequestFromForm(form);

    expect(result.valid).toBe(true);
    if (!result.valid) {
      return;
    }
    expect(result.value.entrypoint).toBe('  /docker-entrypoint.sh  ');
    expect(result.value.command).toBe('  nginx  ');
    expect(result.value.env).toEqual([{ key: ' TOKEN ', value: ' ${TOKEN} ' }]);
    expect(result.value.mounts[0]).toMatchObject({
      content: ' key = value ',
      mode: '0644',
      ignore_if_exists: false,
    });
  });

  it('rejects managed mount sources without the explicit compose prefix', () => {
    const form = emptyComponentForm();
    form.mounts.push({
      source_type: 'controlled_file',
      source: 'config/app.conf',
      target: '/etc/app.conf',
      read_only: true,
      source_is_host_path: false,
      content: 'key=value',
      mode: '0644',
      ignore_if_exists: false,
    });

    expect(componentMountsRequestFromForm(form)).toEqual({ valid: false, error: 'mounts' });
  });

  it('accepts absolute paths for non-volume mounts', () => {
    const form = emptyComponentForm();
    form.mounts.push({
      source_type: 'controlled_file',
      source: '/etc/app/app.env',
      target: '/app/.env',
      read_only: true,
      source_is_host_path: false,
      content: 'key=value',
      mode: '0644',
      ignore_if_exists: false,
    });

    expect(componentMountsRequestFromForm(form)).toMatchObject({
      valid: true,
      value: { mounts: [{ source: '/etc/app/app.env' }] },
    });
  });

  it('rejects backslash mount paths', () => {
    const form = emptyComponentForm();
    form.mounts.push({
      source_type: 'directory',
      source: 'C:\\data',
      target: '/var/lib/app',
      read_only: false,
      source_is_host_path: false,
      content: '',
      mode: '',
      ignore_if_exists: false,
    });

    expect(componentMountsRequestFromForm(form)).toEqual({ valid: false, error: 'mounts' });
  });

  it('rejects bare paths for directory mounts because Compose treats them as volumes', () => {
    const form = emptyComponentForm();
    form.mounts.push({
      source_type: 'directory',
      source: 'data',
      target: '/var/lib/app',
      read_only: false,
      source_is_host_path: false,
      content: '',
      mode: '',
      ignore_if_exists: false,
    });

    expect(componentMountsRequestFromForm(form)).toEqual({ valid: false, error: 'mounts' });
  });

  it('rejects incomplete rows instead of dropping them', () => {
    const form = emptyComponentForm();
    form.name = 'api';
    form.image = 'nginx:1.27';
    form.ports.push({ host_port: '8080', container_port: '' });

    expect(componentRequestFromForm(form)).toEqual({ valid: false, error: 'ports' });
  });

  it('serializes a group without validating other groups', () => {
    const form = emptyComponentForm();
    form.name = 'api';
    form.image = 'nginx:1.27';
    form.ports.push({ host_port: '8080', container_port: '' });

    expect(componentBasicRequestFromForm(form)).toEqual({
      valid: true,
      value: {
        name: 'api',
        image: 'nginx:1.27',
        entrypoint: '',
        command: '',
        pull_policy: 'missing',
        restart_policy: 'unless-stopped',
      },
    });
  });

  it('defaults a new component pull policy to missing', () => {
    const form = emptyComponentForm();
    form.name = 'api';
    form.image = 'nginx:1.27';

    expect(componentBasicRequestFromForm(form)).toEqual({
      valid: true,
      value: {
        name: 'api',
        image: 'nginx:1.27',
        entrypoint: '',
        command: '',
        pull_policy: 'missing',
        restart_policy: 'unless-stopped',
      },
    });
  });

  it('uses the missing default in a new component create request', () => {
    const form = emptyComponentForm();
    form.name = 'api';
    form.image = 'nginx:1.27';

    expect(componentCreateRequestFromForm(form)).toEqual({
      valid: true,
      value: {
        name: 'api',
        image: 'nginx:1.27',
        entrypoint: '',
        command: '',
        pull_policy: 'missing',
        restart_policy: 'unless-stopped',
      },
    });
  });

  it('rejects an omitted or unsupported pull policy', () => {
    const form = emptyComponentForm();
    form.name = 'api';
    form.image = 'nginx:1.27';

    form.pull_policy = '';
    expect(componentBasicRequestFromForm(form)).toEqual({ valid: false, error: 'pullPolicy' });

    form.pull_policy = 'on-demand';
    expect(componentBasicRequestFromForm(form)).toEqual({ valid: false, error: 'pullPolicy' });
  });

  it('rejects an omitted or unsupported restart policy', () => {
    const form = emptyComponentForm();
    form.name = 'api';
    form.image = 'nginx:1.27';

    form.restart_policy = '';
    expect(componentBasicRequestFromForm(form)).toEqual({ valid: false, error: 'restartPolicy' });

    form.restart_policy = 'on-demand';
    expect(componentBasicRequestFromForm(form)).toEqual({ valid: false, error: 'restartPolicy' });
  });

  it('keeps a health check as one command field', () => {
    const form = emptyComponentForm();
    form.healthcheck_enabled = true;
    form.healthcheck_test_mode = 'CMD-SHELL';
    form.healthcheck_test = 'wget -qO- http://localhost/health || exit 1';

    expect(componentRuntimeRequestFromForm(form)).toEqual({
      valid: true,
      value: {
        healthcheck: {
          test_mode: 'CMD-SHELL',
          test: 'wget -qO- http://localhost/health || exit 1',
          interval: undefined,
          timeout: undefined,
          retries: undefined,
          start_period: undefined,
          start_interval: undefined,
          disabled: false,
        },
      },
    });
  });

  it('serializes each network and storage card independently', () => {
    const form = emptyComponentForm();
    form.ports.push({ host_port: '8080', container_port: '80' });
    form.env.push({ key: 'TOKEN', value: '${TOKEN}' });
    form.mounts.push({
      source_type: 'directory',
      source: './data',
      target: '/var/lib/app',
      read_only: false,
      source_is_host_path: false,
      content: '',
      mode: '',
      ignore_if_exists: false,
    });
    form.dependencies.push({ name: 'database', condition: 'service_healthy' });

    expect(componentEndpointsRequestFromForm(form)).toEqual({
      valid: true,
      value: {
        endpoints: [{ protocol: 'tcp', container_port: 80, mode: 'host', listen_port: 8080 }],
      },
    });
    expect(componentEnvRequestFromForm(form)).toEqual({
      valid: true,
      value: { env: [{ key: 'TOKEN', value: '${TOKEN}' }] },
    });
    expect(componentMountsRequestFromForm(form)).toEqual({
      valid: true,
      value: {
        mounts: [
          {
            source_type: 'directory',
            source: './data',
            target: '/var/lib/app',
            read_only: false,
            source_is_host_path: false,
            content: undefined,
            mode: '',
            ignore_if_exists: false,
          },
        ],
      },
    });
    expect(componentDependenciesRequestFromForm(form)).toEqual({
      valid: true,
      value: { dependencies: [{ name: 'database', condition: 'service_healthy' }] },
    });
  });

  it('serializes a gateway HTTP endpoint without a host port', () => {
    const form = emptyComponentForm();
    form.ports.push({
      protocol: 'http',
      host_port: '',
      container_port: '80',
      mode: 'gateway',
      entrypoint: 'web',
      path_prefix: '/',
    });

    expect(componentEndpointsRequestFromForm(form)).toEqual({
      valid: true,
      value: {
        endpoints: [
          {
            protocol: 'http',
            container_port: 80,
            mode: 'gateway',
            entrypoint: 'web',
            path_prefix: '/',
          },
        ],
      },
    });
  });

  it('omits HTTP routing settings from TCP endpoints', () => {
    const form = emptyComponentForm();
    form.ports.push({
      protocol: 'tcp',
      host_port: '5432',
      container_port: '5432',
      mode: 'host',
      bind_address: '127.0.0.1',
      entrypoint: 'web',
      path_prefix: '/database',
    });

    expect(componentEndpointsRequestFromForm(form)).toEqual({
      valid: true,
      value: {
        endpoints: [
          {
            protocol: 'tcp',
            container_port: 5432,
            mode: 'host',
            listen_port: 5432,
            bind_address: '127.0.0.1',
          },
        ],
      },
    });
  });

  it('serializes resources with advanced configuration', () => {
    const form = emptyComponentForm();
    form.resources.limit_memory = '512m';

    expect(componentAdvancedRequestFromForm(form)).toEqual({
      valid: true,
      value: {
        resources: { limit_memory: '512m' },
        tmpfs: [],
        ulimits: [],
      },
    });
  });

  it('serializes advanced cards independently', () => {
    const form = emptyComponentForm();
    form.resources.limit_memory = '512m';
    form.tmpfs.push({ target: '/tmp', size_bytes: '1048576', mode: '1777' });
    form.ulimits.push({ name: 'nofile', soft: '1024', hard: '2048' });

    expect(componentResourcesRequestFromForm(form.resources)).toEqual({
      limit_memory: '512m',
    });
    expect(componentTmpfsRequestFromForm(form.tmpfs)).toEqual({
      valid: true,
      value: [{ target: '/tmp', size_bytes: 1048576, mode: '1777' }],
    });
    expect(componentUlimitsRequestFromForm(form.ulimits)).toEqual({
      valid: true,
      value: [{ name: 'nofile', soft: 1024, hard: 2048 }],
    });
  });
});
