import { describe, expect, it } from 'vitest';
import {
  applyEnvironmentTargetType,
  emptyEnvironmentForm,
  environmentUpdateRequestFromForm,
  initializationEnvironmentRequestFromForm,
  validateEnvironmentForm,
} from './environmentForm';

const messages = {
  host: 'host',
  port: 'port',
  username: 'username',
  workspaceRoot: 'workspace',
};

describe('environmentForm', () => {
  it('builds a local initialization request', () => {
    const form = emptyEnvironmentForm();
    form.workspaceRoot = '/srv/orbit';

    expect(initializationEnvironmentRequestFromForm(form)).toEqual({
      target_type: 'local',
      local: { workspace_root: '/srv/orbit' },
    });
  });

  it('builds an ssh update request with state', () => {
    const form = emptyEnvironmentForm();
    form.targetType = 'ssh';
    form.host = '192.0.2.10';
    form.username = 'deploy';
    form.workspaceRoot = '/srv/orbit';

    expect(environmentUpdateRequestFromForm(form)).toEqual({
      state: 'active',
      target_type: 'ssh',
      ssh: {
        platform: 'linux',
        host: '192.0.2.10',
        port: 22,
        username: 'deploy',
        workspace_root: '/srv/orbit',
      },
    });
  });

  it('rejects an incomplete ssh target', () => {
    const form = emptyEnvironmentForm();
    form.targetType = 'ssh';

    expect(validateEnvironmentForm(form, messages)).toMatchObject({
      host: 'host',
      username: 'username',
    });
  });

  it('accepts a local home workspace root as written', () => {
    const form = emptyEnvironmentForm();
    form.workspaceRoot = '~/.pomelo-orbit';

    expect(validateEnvironmentForm(form, messages).workspaceRoot).toBe('');
    expect(initializationEnvironmentRequestFromForm(form)).toEqual({
      target_type: 'local',
      local: { workspace_root: '~/.pomelo-orbit' },
    });
  });

  it('switches to ssh with a platform workspace root', () => {
    const form = emptyEnvironmentForm();
    form.workspaceRoot = 'C:\\orbit';

    expect(applyEnvironmentTargetType(form, 'ssh', undefined, '/srv/orbit').workspaceRoot).toBe(
      '~/.pomelo-orbit'
    );
  });

  it('prefills windows ssh host and username from control-plane display', () => {
    const form = emptyEnvironmentForm();
    const next = applyEnvironmentTargetType(
      form,
      'ssh',
      {
        platform: 'windows',
        host: 'DESKTOP-ORBIT',
        username: 'orbit',
        workspace_root: '~/.pomelo-orbit',
      },
      '~/.pomelo-orbit'
    );

    expect(next).toMatchObject({
      targetType: 'ssh',
      platform: 'windows',
      host: 'DESKTOP-ORBIT',
      username: 'orbit',
      workspaceRoot: '~/.pomelo-orbit',
    });
  });
});
