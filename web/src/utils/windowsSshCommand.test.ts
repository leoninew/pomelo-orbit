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

    expect(command).toMatch(/^\$encoded = '[A-Za-z0-9+/=]+'; \$process = Start-Process/);
    expect(command).toContain('-Verb RunAs');
    expect(command).toContain("'-EncodedCommand', $encoded");
    expect(command).toContain('-Wait -PassThru');
    const encoded = command.match(/^\$encoded = '([^']+)'/)?.[1];
    expect(encoded).toBeTruthy();
    if (!encoded) {
      throw new Error('encoded PowerShell script is missing');
    }
    const bytes = Uint8Array.from(atob(encoded), (char) => char.charCodeAt(0));
    const script = String.fromCharCode(
      ...Array.from(
        { length: bytes.length / 2 },
        (_, index) => bytes[index * 2] | (bytes[index * 2 + 1] << 8)
      )
    );
    expect(script).toContain("$key = 'ssh-ed25519 AAAAorbit''s-key'");
    expect(script).toContain("$workspace = 'C:\\Users\\orbit\\.pomelo-orbit'");
    expect(script).not.toContain('ssh.exe');
    expect(script).toContain('Invoke-Expression $setupScript');
    expect(script).toContain('administrators_authorized_keys');
    expect(script).toContain("Add-WindowsCapability -Online -Name 'OpenSSH.Server~~~~0.0.1.0'");
    expect(script).toContain("$sshdConfig = Join-Path $env:ProgramData 'ssh\\sshd_config'");
    expect(script).toContain('ListenAddress 0.0.0.0');
    expect(script).toContain('ListenAddress ::');
    expect(script).toContain(
      "$sshKeygenExe = Join-Path $env:WINDIR 'System32\\OpenSSH\\ssh-keygen.exe'"
    );
    expect(script).toContain('$sshKeygenExe -A');
    expect(script).toContain('Restart-Service -Name sshd -Force');
    expect(script).toContain('New-NetFirewallRule -Name $firewallRuleName');
    expect(script).toContain('Get-NetTCPConnection -LocalPort 2222 -State Listen');
    expect(script).toContain('docker-desktop');
    expect(script).toContain('docker.exe compose version');
    expect(script).toContain("$serverOS -ne 'linux'");
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
