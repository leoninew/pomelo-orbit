import { describe, expect, it } from 'vitest';
import { buildLinuxSshInitializationCommand } from '@/utils/linuxSshCommand';

describe('linuxSshCommand', () => {
  it('builds an idempotent command for the current SSH user', () => {
    const script = buildLinuxSshInitializationCommand({
      publicKey: "ssh-ed25519 AAAAorbit's-key",
    });

    expect(script).toContain('set -eu');
    expect(script).toContain('mkdir -p "$HOME/.ssh"');
    expect(script).toContain('chmod 700 "$HOME/.ssh"');
    expect(script).toContain('chmod 600 "$HOME/.ssh/authorized_keys"');
    expect(script).toContain("grep -qxF -- 'ssh-ed25519 AAAAorbit'\"'\"'s-key'");
    expect(script).toContain("printf '%s\\n' 'ssh-ed25519 AAAAorbit'\"'\"'s-key'");
    expect(script).not.toContain('sudo');
    expect(script).not.toContain('sshd');
    expect(script).not.toContain('docker');
  });

  it('does not produce a command without a deployment key', () => {
    expect(buildLinuxSshInitializationCommand({ publicKey: '  ' })).toBe('');
  });
});
