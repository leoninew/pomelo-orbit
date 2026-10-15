import { describe, expect, it } from 'vitest';
import { buildWindowsSshInitializationCommand } from '@/utils/windowsSshCommand';

describe('windowsSshCommand', () => {
  it('builds a PowerShell command from the configured SSH target', () => {
    const script = buildWindowsSshInitializationCommand({
      host: 'DESKTOP-ORBIT',
      port: 2222,
      username: 'Administrator',
      workspaceRoot: `C:\\Users\\orbit\\.pomelo-orbit`,
      publicKey: "ssh-ed25519 AAAAorbit's-key",
      keyComment: "# Pomelo Orbit | project=Orbit's project",
    });

    expect(script.startsWith('& {\n')).toBe(true);
    expect(script.endsWith('\n}\n')).toBe(true);
    expect(script).toContain('WindowsBuiltInRole]::Administrator');
    expect(script).toContain("Write-Host 'Target Administrator@DESKTOP-ORBIT:2222'");
    expect(script).toContain("$key = 'ssh-ed25519 AAAAorbit''s-key'");
    expect(script).toContain("$targetUsername = 'Administrator'");
    expect(script).toContain("$keyComment = '# Pomelo Orbit | project=Orbit''s project'");
    expect(script).toContain('if ($line -ceq $key)');
    expect(script).toContain('$updated.Add($keyComment)');
    expect(script).toContain('$updated.RemoveAt($updated.Count - 1)');
    expect(script).toContain('[System.Text.UTF8Encoding]::new($false)');
    expect(script).toContain("$workspace = 'C:\\Users\\orbit\\.pomelo-orbit'");
    expect(script).toContain('Get-CimInstance -ClassName Win32_UserProfile');
    expect(script).toContain("$authorizedKeys = Join-Path $sshDir 'authorized_keys'");
    expect(script).toContain('Add-DeploymentKey $authorizedKeys');
    expect(script).toContain('administrators_authorized_keys');
    expect(script).toContain('Add-DeploymentKey $administratorsAuthorizedKeys');
    expect(script).not.toContain("Join-Path $env:USERPROFILE '.ssh'");
    expect(script).toContain("Add-WindowsCapability -Online -Name 'OpenSSH.Server~~~~0.0.1.0'");
    expect(script).toContain("$programDataSsh = Join-Path $env:ProgramData 'ssh'");
    expect(script).toContain("$sshdConfig = Join-Path $programDataSsh 'sshd_config'");
    expect(script).toContain('sshd_config_default');
    expect(script).toContain("'Port 2222', 'PubkeyAuthentication yes'");
    expect(script).toContain('function Test-OrbitSshListening');
    expect(script).toContain('Start-Service -Name sshd');
    expect(script).toContain(
      "$sshKeygenExe = Join-Path $env:WINDIR 'System32\\OpenSSH\\ssh-keygen.exe'"
    );
    expect(script).toContain('$sshKeygenExe -A');
    expect(script).toContain(
      "Get-ChildItem -LiteralPath $programDataSsh -Filter 'ssh_host_*_key' -File"
    );
    expect(script).toContain("SetSecurityDescriptorSddlForm('O:BAD:P(A;;FA;;;SY)(A;;FA;;;BA)')");
    expect(script).toContain('Set-Acl -LiteralPath $_.FullName -AclObject $hostKeyAcl');
    expect(script.indexOf('Set-Acl -LiteralPath $_.FullName')).toBeLessThan(
      script.indexOf('& $sshdExe -t -f $sshdConfig')
    );
    expect(script).toContain('Restart-Service -Name sshd -Force');
    expect(script).toContain("LogName = @('OpenSSH/Admin', 'OpenSSH/Operational')");
    expect(script).toContain("ProviderName = 'Service Control Manager'");
    expect(script).toContain('ExitCode,ServiceSpecificExitCode');
    expect(script).toContain('New-NetFirewallRule -Name $firewallRuleName');
    expect(script).toContain('-Profile Any');
    expect(script).toContain('Test-OrbitSshListening 2222');
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
        keyComment: "# Pomelo Orbit | project=Orbit's project",
      })
    ).toBe('');
  });
});
