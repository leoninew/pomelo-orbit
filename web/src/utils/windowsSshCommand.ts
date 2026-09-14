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

function encodePowerShellScript(value: string): string {
  const bytes = new Uint8Array(value.length * 2);
  for (let index = 0; index < value.length; index += 1) {
    const code = value.charCodeAt(index);
    bytes[index * 2] = code & 0xff;
    bytes[index * 2 + 1] = code >> 8;
  }
  let binary = '';
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary);
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

  const script = `$ErrorActionPreference = 'Stop'
$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = [Security.Principal.WindowsPrincipal]::new($identity)
$isAdministrator = $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdministrator) {
  throw 'Run this command in an elevated PowerShell window (Run as Administrator).'
}
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
$sshdConfig = Join-Path $env:ProgramData 'ssh\\sshd_config'
if (-not (Test-Path -LiteralPath $sshdConfig)) {
  throw 'OpenSSH Server configuration file was not found.'
}
$sshdConfigLines = [string[]](Get-Content -LiteralPath $sshdConfig)
$matchIndex = -1
for ($index = 0; $index -lt $sshdConfigLines.Length; $index++) {
  if ($sshdConfigLines[$index] -match '^\\s*Match\\s+') {
    $matchIndex = $index
    break
  }
}
if ($matchIndex -lt 0) {
  throw 'OpenSSH Server configuration has no global Match boundary.'
}
$globalLines = @($sshdConfigLines[0..($matchIndex - 1)] | Where-Object { $_ -notmatch '^\\s*(?:#?\\s*)?(?:Port|ListenAddress)\\b' })
$matchLines = @($sshdConfigLines[$matchIndex..($sshdConfigLines.Length - 1)])
$sshdConfigLines = @('Port ${port}', 'ListenAddress 0.0.0.0', 'ListenAddress ::') + $globalLines + $matchLines
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
if ($sshd.Status -eq 'Running') {
  Restart-Service -Name sshd -Force
} else {
  Start-Service -Name sshd
}
$firewallRuleName = 'Pomelo-Orbit-OpenSSH-In-TCP'
$firewallRule = Get-NetFirewallRule -Name $firewallRuleName -ErrorAction SilentlyContinue
if ($firewallRule) {
  $firewallRule | Set-NetFirewallRule -Enabled True
  $firewallRule | Get-NetFirewallPortFilter | Set-NetFirewallPortFilter -LocalPort ${port}
} else {
  New-NetFirewallRule -Name $firewallRuleName -DisplayName 'Pomelo Orbit OpenSSH Server' -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort ${port} | Out-Null
}
for ($attempt = 0; $attempt -lt 10; $attempt++) {
  if ((Get-NetTCPConnection -LocalPort ${port} -State Listen -ErrorAction SilentlyContinue)) {
    break
  }
  Start-Sleep -Milliseconds 500
}
if (-not (Get-NetTCPConnection -LocalPort ${port} -State Listen -ErrorAction SilentlyContinue)) {
  throw 'OpenSSH Server did not start listening on the configured port.'
}
Write-Host '[0/3] Windows OpenSSH Server is installed, running, and reachable on the configured port.'
Write-Host ${powershellSingleQuote(`Target ${username}@${host}:${port}`)}
$setupScript = @'
$ErrorActionPreference = 'Stop'
$key = ${powershellSingleQuote(key)}
$workspace = ${powershellSingleQuote(workspace)}
$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = [Security.Principal.WindowsPrincipal]::new($identity)
$isAdministrator = $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if ($isAdministrator) {
  $authorizedKeys = Join-Path $env:ProgramData 'ssh\\administrators_authorized_keys'
  New-Item -ItemType Directory -Force -Path (Split-Path $authorizedKeys) | Out-Null
} else {
  $sshDir = Join-Path $env:USERPROFILE '.ssh'
  $authorizedKeys = Join-Path $sshDir 'authorized_keys'
  New-Item -ItemType Directory -Force -Path $sshDir | Out-Null
}
if (-not (Test-Path -LiteralPath $authorizedKeys) -or -not (Select-String -LiteralPath $authorizedKeys -SimpleMatch -Quiet -Pattern $key)) {
  Add-Content -LiteralPath $authorizedKeys -Value $key
}
if ($isAdministrator) {
  icacls $authorizedKeys /inheritance:r /grant:r '*S-1-5-32-544:F' /grant:r '*S-1-5-18:F' | Out-Null
}
New-Item -ItemType Directory -Force -Path $workspace | Out-Null
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
'@
Invoke-Expression $setupScript
`;

  const encodedScript = encodePowerShellScript(script);
  return `$encoded = '${encodedScript}'; $process = Start-Process -FilePath powershell.exe -Verb RunAs -ArgumentList @('-NoProfile', '-NonInteractive', '-ExecutionPolicy', 'Bypass', '-EncodedCommand', $encoded) -Wait -PassThru; if ($process.ExitCode -ne 0) { exit $process.ExitCode }`;
}
