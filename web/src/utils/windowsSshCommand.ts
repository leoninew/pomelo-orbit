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
  if (!key) return '';

  const host = input.host.trim() || '<host>';
  const username = input.username.trim() || '<ssh-user>';
  const port = Number.isInteger(input.port) ? input.port : 22;
  const workspace = input.workspaceRoot.trim() || 'C:\\Users\\<user>\\.pomelo-orbit';

  return `$remoteScript = @'
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
$remoteScript | ssh -p ${port} ${powershellSingleQuote(`${username}@${host}`)} powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -Command -`;
}
