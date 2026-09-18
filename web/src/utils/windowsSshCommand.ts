export type WindowsSshCommandInput = {
  host: string;
  port: number;
  username: string;
  workspaceRoot: string;
  publicKey: string;
};

function powershellSingleQuote(value: string): string {
  return `'${value.replace(/'/g, "''")}'`;
}

export function buildWindowsSshInitializationCommand(input: WindowsSshCommandInput): string {
  const key = input.publicKey.trim();
  if (!key) {
    return '';
  }

  const host = input.host.trim() || '<host>';
  const username = input.username.trim() || '<ssh-user>';
  const port = Number.isInteger(input.port) ? input.port : 22;
  const workspace = input.workspaceRoot.trim() || 'C:\\Users\\<user>\\.pomelo-orbit';

  return `$ErrorActionPreference = 'Stop'
$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = [Security.Principal.WindowsPrincipal]::new($identity)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
  throw 'Run this command in an elevated PowerShell window (Run as Administrator).'
}
Write-Host ${powershellSingleQuote(`Target ${username}@${host}:${port}`)}
$sshClientCapability = Get-WindowsCapability -Online -Name 'OpenSSH.Client~~~~0.0.1.0'
if ($sshClientCapability.State -ne 'Installed') {
  Add-WindowsCapability -Online -Name 'OpenSSH.Client~~~~0.0.1.0' | Out-Null
}
$sshServerCapability = Get-WindowsCapability -Online -Name 'OpenSSH.Server~~~~0.0.1.0'
if ($sshServerCapability.State -ne 'Installed') {
  Add-WindowsCapability -Online -Name 'OpenSSH.Server~~~~0.0.1.0' | Out-Null
}
$sshd = Get-Service -Name sshd -ErrorAction SilentlyContinue
if (-not $sshd) {
  throw 'OpenSSH Server installation did not provide the sshd service.'
}
$programDataSsh = Join-Path $env:ProgramData 'ssh'
New-Item -ItemType Directory -Force -Path $programDataSsh | Out-Null
$sshdConfig = Join-Path $programDataSsh 'sshd_config'
if (-not (Test-Path -LiteralPath $sshdConfig)) {
  $openSshDirectory = Join-Path $env:WINDIR 'System32\\OpenSSH'
  $sshdConfigTemplate = Join-Path $openSshDirectory 'sshd_config_default'
  if (-not (Test-Path -LiteralPath $sshdConfigTemplate)) {
    $sshdConfigTemplate = Join-Path $openSshDirectory 'sshd_config'
  }
  if (Test-Path -LiteralPath $sshdConfigTemplate) {
    Copy-Item -LiteralPath $sshdConfigTemplate -Destination $sshdConfig
  } else {
    @(
      'PubkeyAuthentication yes',
      'AuthorizedKeysFile .ssh/authorized_keys',
      'Subsystem sftp sftp-server.exe',
      'Match Group administrators',
      '    AuthorizedKeysFile __PROGRAMDATA__/ssh/administrators_authorized_keys'
    ) | Set-Content -LiteralPath $sshdConfig -Encoding ascii
  }
}
$sshdConfigLines = [string[]](Get-Content -LiteralPath $sshdConfig)
$matchIndex = -1
for ($index = 0; $index -lt $sshdConfigLines.Length; $index++) {
  if ($sshdConfigLines[$index] -match '^\\s*Match\\s+') {
    $matchIndex = $index
    break
  }
}
$keepLine = { $_ -notmatch '^\\s*(?:#?\\s*)?(?:Port|ListenAddress|PubkeyAuthentication)\\b' }
if ($matchIndex -lt 0) {
  $globalLines = @($sshdConfigLines | Where-Object $keepLine)
  $matchLines = @()
} else {
  $globalLines = @($sshdConfigLines[0..($matchIndex - 1)] | Where-Object $keepLine)
  $matchLines = @($sshdConfigLines[$matchIndex..($sshdConfigLines.Length - 1)])
}
$sshdConfigLines = @('Port ${port}', 'PubkeyAuthentication yes') + $globalLines + $matchLines
Set-Content -LiteralPath $sshdConfig -Value $sshdConfigLines -Encoding ascii
$sshKeygenExe = Join-Path $env:WINDIR 'System32\\OpenSSH\\ssh-keygen.exe'
& $sshKeygenExe -A | Out-Null
if ($LASTEXITCODE -ne 0) {
  throw 'OpenSSH Server host keys could not be generated.'
}
$sshdExe = Join-Path $env:WINDIR 'System32\\OpenSSH\\sshd.exe'
& $sshdExe -t -f $sshdConfig
if ($LASTEXITCODE -ne 0) {
  throw 'OpenSSH Server configuration is invalid after setting the configured port.'
}
Set-Service -Name sshd -StartupType Automatic
$sshd = Get-Service -Name sshd
if ($sshd.Status -eq 'Running') {
  Restart-Service -Name sshd -Force
} else {
  Start-Service -Name sshd
}
$sshd = Get-Service -Name sshd
if ($sshd.Status -ne 'Running') {
  throw 'OpenSSH Server service failed to stay running. Check C:\\ProgramData\\ssh\\logs and the System event log.'
}
$firewallRuleName = 'Pomelo-Orbit-OpenSSH-In-TCP'
$firewallRule = Get-NetFirewallRule -Name $firewallRuleName -ErrorAction SilentlyContinue
if ($firewallRule) {
  $firewallRule | Set-NetFirewallRule -Enabled True -Profile Any
  $firewallRule | Get-NetFirewallPortFilter | Set-NetFirewallPortFilter -LocalPort ${port}
} else {
  New-NetFirewallRule -Name $firewallRuleName -DisplayName 'Pomelo Orbit OpenSSH Server' -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort ${port} -Profile Any | Out-Null
}
function Test-OrbitSshListening([int]$Port) {
  $previous = $ErrorActionPreference
  $ErrorActionPreference = 'SilentlyContinue'
  try {
    return @(Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue).Count -gt 0
  } finally {
    $ErrorActionPreference = $previous
  }
}
for ($attempt = 0; $attempt -lt 10; $attempt++) {
  if (Test-OrbitSshListening ${port}) {
    break
  }
  Start-Sleep -Milliseconds 500
}
if (-not (Test-OrbitSshListening ${port})) {
  throw "OpenSSH Server did not start listening on port ${port}."
}
Write-Host '[0/3] Windows OpenSSH Server is installed, running, and reachable on the configured port.'
$key = ${powershellSingleQuote(key)}
$targetUsername = ${powershellSingleQuote(username)}
$workspace = ${powershellSingleQuote(workspace)}
try {
  $targetAccount = [Security.Principal.NTAccount]::new($targetUsername)
  $targetSid = $targetAccount.Translate([Security.Principal.SecurityIdentifier]).Value
} catch {
  throw "Configured SSH user '$targetUsername' could not be resolved to a Windows account."
}
$targetProfile = Get-CimInstance -ClassName Win32_UserProfile -Filter "SID='$targetSid'" | Select-Object -First 1
if (-not $targetProfile -or [string]::IsNullOrWhiteSpace($targetProfile.LocalPath)) {
  throw "Configured SSH user '$targetUsername' does not have a local Windows profile."
}
function Add-DeploymentKey([string]$path) {
  New-Item -ItemType Directory -Force -Path (Split-Path $path) | Out-Null
  if (-not (Test-Path -LiteralPath $path) -or -not (Select-String -LiteralPath $path -SimpleMatch -Quiet -Pattern $key)) {
    Add-Content -LiteralPath $path -Value $key
  }
}
$sshDir = Join-Path $targetProfile.LocalPath '.ssh'
$authorizedKeys = Join-Path $sshDir 'authorized_keys'
Add-DeploymentKey $authorizedKeys
$targetDirectoryAcl = '*' + $targetSid + ':(OI)(CI)F'
$targetFileAcl = '*' + $targetSid + ':F'
icacls $sshDir /inheritance:r /grant:r $targetDirectoryAcl /grant:r '*S-1-5-18:(OI)(CI)F' | Out-Null
icacls $authorizedKeys /inheritance:r /grant:r $targetFileAcl /grant:r '*S-1-5-18:F' | Out-Null
$administratorsAuthorizedKeys = Join-Path $env:ProgramData 'ssh\\administrators_authorized_keys'
Add-DeploymentKey $administratorsAuthorizedKeys
icacls $administratorsAuthorizedKeys /inheritance:r /grant:r '*S-1-5-32-544:F' /grant:r '*S-1-5-18:F' | Out-Null
New-Item -ItemType Directory -Force -Path $workspace | Out-Null
$workspaceAcl = '*' + $targetSid + ':(OI)(CI)M'
icacls $workspace /grant:r $workspaceAcl /grant:r '*S-1-5-18:(OI)(CI)F' | Out-Null
Write-Host '[1/3] Orbit deployment key and workspace are configured.'
Write-Host '[2/3] Checking WSL2 and Docker Desktop...'
$distros = (& wsl.exe -l -v | Out-String) -replace [char]0, ''
if ($LASTEXITCODE -ne 0 -or $distros -notmatch '(?m)^\\s*\\*?\\s*docker-desktop\\s+\\S+\\s+2\\s*$') {
  throw 'WSL2 Docker Desktop distribution is unavailable'
}
& docker.exe version --format '{{.Server.Version}}' | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Docker Desktop engine is unavailable' }
& docker.exe compose version | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Docker Compose is unavailable' }
$serverOS = (& docker.exe version --format '{{.Server.Os}}').Trim()
if ($LASTEXITCODE -ne 0 -or $serverOS -ne 'linux') { throw 'Docker Desktop must use Linux containers' }
& docker.exe info | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Docker daemon is unavailable' }
Write-Host '[3/3] Docker Desktop and Docker Compose are ready.'
`;
}
