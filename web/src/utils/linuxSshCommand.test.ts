import { describe, expect, it } from 'vitest';
import { buildLinuxSshInitializationCommand } from '@/utils/linuxSshCommand';

describe('linuxSshCommand', () => {
  it('authorizes the SSH user and prepares the configured workspace', () => {
    const script = buildLinuxSshInitializationCommand({
      username: 'deploy',
      workspaceRoot: '/srv/orbit work',
      publicKey: "ssh-ed25519 AAAAorbit's-key",
    });

    expect(script).toContain('set -eu');
    expect(script).toContain('if [ "$(id -un)" != \'deploy\' ]; then');
    expect(script).toContain('mkdir -p "$HOME/.ssh"');
    expect(script).toContain('chmod 700 "$HOME/.ssh"');
    expect(script).toContain('chmod 600 "$HOME/.ssh/authorized_keys"');
    expect(script).toContain("grep -qxF -- 'ssh-ed25519 AAAAorbit'\"'\"'s-key'");
    expect(script).toContain("printf '%s\\n' 'ssh-ed25519 AAAAorbit'\"'\"'s-key'");
    expect(script).toContain("workspace_root='/srv/orbit work'");
    expect(script).toContain('if [ ! -d "$workspace_root" ]; then');
    expect(script).toContain('else');
    expect(script).toContain(
      'run_as_admin mkdir -p -- "$workspace_root/pipeline" "$workspace_root/deployment"'
    );
    expect(script).toContain(
      'run_as_admin chown -- "$(id -u):$(id -g)" "$workspace_root" "$workspace_root/pipeline" "$workspace_root/deployment"'
    );
    expect(script).toContain(
      'run_as_admin chown -- "$(id -u):$(id -g)" "$workspace_root/pipeline" "$workspace_root/deployment"'
    );
    expect(script).not.toContain('chown -R');
    expect(script).not.toContain('sshd');
    expect(script).not.toContain('docker');
  });

  it('expands the remote SSH user home directory without interpolating the path into shell code', () => {
    const script = buildLinuxSshInitializationCommand({
      username: 'deploy',
      workspaceRoot: "~/orbit's workspace",
      publicKey: 'ssh-ed25519 AAAAorbit',
    });

    expect(script).toContain("workspace_root=\"$HOME\"/'orbit'\"'\"'s workspace'");
    expect(
      buildLinuxSshInitializationCommand({
        username: 'deploy',
        workspaceRoot: '~',
        publicKey: 'ssh-ed25519 AAAAorbit',
      })
    ).toContain('workspace_root="$HOME"');
  });

  it('does not produce a command without a deployment key or target', () => {
    expect(
      buildLinuxSshInitializationCommand({
        username: 'deploy',
        workspaceRoot: '/srv/orbit',
        publicKey: '  ',
      })
    ).toBe('');
    expect(
      buildLinuxSshInitializationCommand({
        username: '',
        workspaceRoot: '/srv/orbit',
        publicKey: 'key',
      })
    ).toBe('');
  });
});
