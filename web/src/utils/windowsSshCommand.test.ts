import { describe, expect, it } from 'vitest';
import { buildWindowsSshInitializationCommand } from '@/utils/windowsSshCommand';

describe('windowsSshCommand', () => {
  it('builds a PowerShell command from the configured SSH target', () => {
    const command = buildWindowsSshInitializationCommand({
      host: 'DESKTOP-ORBIT',
      port: 2222,
      username: 'Administrator',
      workspaceRoot: `C:\\Users\\orbit\\.pomelo-orbit`,
      publicKey: "ssh-ed25519 AAAAorbit's-key",
    });

    expect(command).toContain("$key = 'ssh-ed25519 AAAAorbit''s-key'");
    expect(command).toContain("$workspace = 'C:\\Users\\orbit\\.pomelo-orbit'");
    expect(command).toContain("ssh -p 2222 'Administrator@DESKTOP-ORBIT'");
    expect(command).toContain('administrators_authorized_keys');
    expect(command).toContain('docker-desktop');
    expect(command).toContain('docker.exe compose version');
    expect(command).toContain("$serverOS -ne 'linux'");
  });

  it('does not produce a command without a deployment key', () => {
    expect(
      buildWindowsSshInitializationCommand({
        host: 'host',
        port: 22,
        username: 'user',
        workspaceRoot: 'C:\\orbit',
        publicKey: '  ',
      })
    ).toBe('');
  });
});
