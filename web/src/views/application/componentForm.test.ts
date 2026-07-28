import { describe, expect, it } from 'vitest';
import { componentRequestFromForm, emptyComponentForm } from './componentForm';

describe('componentForm', () => {
  it('preserves entered configuration text', () => {
    const form = emptyComponentForm();
    form.name = 'api';
    form.image = 'nginx:1.27';
    form.command.push({ value: '  nginx  ' });
    form.env.push({ key: ' TOKEN ', value: ' ${TOKEN} ' });
    form.mounts.push({
      source_type: 'file',
      source: 'config/app.conf',
      target: '/etc/app.conf',
      read_only: true,
      content: ' key = value ',
      content_mode: 'sync',
    });

    const result = componentRequestFromForm(form);

    expect(result.valid).toBe(true);
    if (!result.valid) {
      return;
    }
    expect(result.value.command).toEqual(['  nginx  ']);
    expect(result.value.env).toEqual([{ key: ' TOKEN ', value: ' ${TOKEN} ' }]);
    expect(result.value.mounts[0]).toMatchObject({
      content: ' key = value ',
      content_mode: 'sync',
    });
  });

  it('rejects incomplete rows instead of dropping them', () => {
    const form = emptyComponentForm();
    form.name = 'api';
    form.image = 'nginx:1.27';
    form.ports.push({ host_port: '8080', container_port: '' });

    expect(componentRequestFromForm(form)).toEqual({ valid: false, error: 'ports' });
  });
});
