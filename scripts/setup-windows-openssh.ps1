#Requires -Version 5.1
#Requires -RunAsAdministrator

<#
.SYNOPSIS
Configures a Windows OpenSSH Server target for Pomelo Orbit integration tests.

.DESCRIPTION
Run this script from an elevated PowerShell session as the administrator account
that Pomelo Orbit will use for SSH. It installs OpenSSH Server, enables SSH/SFTP
public-key authentication, creates a workspace, and prints Environment values.

The script owns one dedicated, unencrypted Ed25519 client key pair at
%USERPROFILE%\.ssh\pomelo_orbit_ed25519 and
%USERPROFILE%\.ssh\pomelo_orbit_ed25519_pub. It generates the pair on first
run and validates both files match on subsequent runs. The existing sshd_config
is backed up once before being replaced.

.EXAMPLE
.\scripts\setup-windows-openssh.ps1

.EXAMPLE
.\scripts\setup-windows-openssh.ps1 -ListenAddress 0.0.0.0 `
    -EnvironmentHost windows-deploy.example.internal `
    -FirewallRemoteAddress 10.20.0.0/16
#>

[CmdletBinding()]
param(
    [ValidateRange(1, 65535)]
    [int]$Port = 2222,

    [ValidateNotNullOrEmpty()]
    [string]$ListenAddress = '127.0.0.1',

    [string]$EnvironmentHost,

    [ValidateNotNullOrEmpty()]
    [string]$WorkspacePath = (Join-Path $env:USERPROFILE '.pomelo-orbit'),

    [ValidateNotNullOrEmpty()]
    [string[]]$FirewallRemoteAddress = @('LocalSubnet'),

    [ValidateSet('Domain', 'Private', 'Public')]
    [string[]]$FirewallProfile = @('Domain', 'Private'),

    [switch]$ReplaceAuthorizedKeys,
    [switch]$RegenerateHostKey,
    [switch]$SkipConnectionTest
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Invoke-NativeCommand {
    param(
        [Parameter(Mandatory = $true)][string]$FilePath,
        [Parameter(Mandatory = $true)][AllowEmptyString()][string[]]$Arguments
    )

    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$FilePath failed with exit code $LASTEXITCODE"
    }
}

function Invoke-Icacls {
    param([Parameter(Mandatory = $true)][string[]]$Arguments)

    $output = & icacls.exe @Arguments 2>&1
    if ($LASTEXITCODE -ne 0) {
        $reason = ($output -join [Environment]::NewLine).Trim()
        if ([string]::IsNullOrWhiteSpace($reason)) {
            $reason = "icacls exited with code $LASTEXITCODE"
        }
        throw $reason
    }
}

function Set-OpenSSHRestrictedAcl {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [switch]$Directory
    )

    $itemKind = if ($Directory) { 'directory' } else { 'file' }
    try {
        Invoke-Icacls @($Path, '/inheritance:r')
        if ($Directory) {
            Invoke-Icacls @($Path, '/grant:r', '*S-1-5-18:(OI)(CI)F')
            Invoke-Icacls @($Path, '/grant', '*S-1-5-32-544:(OI)(CI)F')
        } else {
            Invoke-Icacls @($Path, '/grant:r', '*S-1-5-18:F')
            Invoke-Icacls @($Path, '/grant', '*S-1-5-32-544:F')
        }
        Invoke-Icacls @($Path, '/setowner', '*S-1-5-32-544')
        $runAsUser = [Security.Principal.WindowsIdentity]::GetCurrent().Name
        Invoke-Icacls @($Path, '/remove:g', $runAsUser)
        Write-Host "Processing $itemKind`: $Path, succeeded"
    } catch {
        Write-Host "Processing $itemKind`: $Path, failed, $($_.Exception.Message)"
        throw
    }
}
function Get-PublicKeyIdentity {
    param([Parameter(Mandatory = $true)][string]$PublicKey)

    $fields = $PublicKey.Trim() -split '\s+'
    for ($index = 0; $index -lt ($fields.Count - 1); $index++) {
        if ($fields[$index] -match '^(ssh-|ecdsa-|sk-)') {
            return "$($fields[$index]) $($fields[$index + 1])"
        }
    }
    return $null
}

function Read-OpenSSHPrivateKeyString {
    param(
        [Parameter(Mandatory = $true)][byte[]]$Bytes,
        [Parameter(Mandatory = $true)][ref]$Offset
    )

    $start = [int]$Offset.Value
    if ($start + 4 -gt $Bytes.Length) {
        throw 'Invalid OpenSSH private key length prefix.'
    }
    $length =
        ([uint32]$Bytes[$start] * 16777216) +
        ([uint32]$Bytes[$start + 1] * 65536) +
        ([uint32]$Bytes[$start + 2] * 256) +
        [uint32]$Bytes[$start + 3]
    $start += 4
    if ($length -gt [uint32]($Bytes.Length - $start)) {
        throw 'Invalid OpenSSH private key string length.'
    }
    $Offset.Value = $start + [int]$length
    return [Text.Encoding]::ASCII.GetString($Bytes, $start, [int]$length)
}

function Assert-UnencryptedOpenSSHPrivateKey {
    param([Parameter(Mandatory = $true)][string]$PrivateKeyPath)

    $content = [IO.File]::ReadAllText($PrivateKeyPath, [Text.Encoding]::ASCII).Trim()
    $header = '-----BEGIN OPENSSH PRIVATE KEY-----'
    $footer = '-----END OPENSSH PRIVATE KEY-----'
    if (-not $content.StartsWith($header) -or -not $content.EndsWith($footer)) {
        throw "Dedicated private key must use the OpenSSH private-key format: $PrivateKeyPath"
    }
    $base64 = ($content.Substring($header.Length, $content.Length - $header.Length - $footer.Length) -replace '\s', '')
    try {
        $bytes = [Convert]::FromBase64String($base64)
    } catch {
        throw "Dedicated private key is not valid base64 OpenSSH data: $PrivateKeyPath"
    }

    $magic = [Text.Encoding]::ASCII.GetBytes("openssh-key-v1`0")
    if ($bytes.Length -lt $magic.Length + 8) {
        throw "Dedicated private key is truncated: $PrivateKeyPath"
    }
    for ($index = 0; $index -lt $magic.Length; $index++) {
        if ($bytes[$index] -ne $magic[$index]) {
            throw "Dedicated private key is not an OpenSSH key-v1 payload: $PrivateKeyPath"
        }
    }

    $offset = $magic.Length
    $offsetReference = [ref]$offset
    $cipherName = Read-OpenSSHPrivateKeyString -Bytes $bytes -Offset $offsetReference
    $kdfName = Read-OpenSSHPrivateKeyString -Bytes $bytes -Offset $offsetReference
    if ($cipherName -ne 'none' -or $kdfName -ne 'none') {
        throw "Dedicated private key must not use a passphrase: $PrivateKeyPath"
    }
}

if ([Environment]::OSVersion.Platform -ne [PlatformID]::Win32NT) {
    throw 'This script only supports Windows.'
}

$parsedListenAddress = $null
if (-not [Net.IPAddress]::TryParse($ListenAddress, [ref]$parsedListenAddress)) {
    throw "ListenAddress must be an IP address, received: $ListenAddress"
}
$isLoopback = [Net.IPAddress]::IsLoopback($parsedListenAddress)

Write-Host 'Checking Windows OpenSSH Client and Server capabilities...'
$capabilityNames = @(
    'OpenSSH.Client~~~~0.0.1.0',
    'OpenSSH.Server~~~~0.0.1.0'
)
foreach ($capabilityName in $capabilityNames) {
    Write-Host "Checking capability: $capabilityName"
    $capability = Get-WindowsCapability -Online -Name $capabilityName
    if ($capability.State -eq 'Installed') {
        Write-Host "Already installed: $capabilityName"
        continue
    }

    Write-Host "Installing $capabilityName through Windows Optional Features. Downloads can take several minutes."
    $installResult = Add-WindowsCapability -Online -Name $capabilityName
    if ($installResult.RestartNeeded) {
        throw "$capabilityName was installed but Windows must restart before configuration can continue."
    }
    Write-Host "Installed: $capabilityName"
}

$openSshDirectory = Join-Path $env:WINDIR 'System32\OpenSSH'
$sshd = Join-Path $openSshDirectory 'sshd.exe'
$ssh = Join-Path $openSshDirectory 'ssh.exe'
$sshKeygen = Join-Path $openSshDirectory 'ssh-keygen.exe'
foreach ($executable in @($sshd, $ssh, $sshKeygen)) {
    if (-not (Test-Path -LiteralPath $executable -PathType Leaf)) {
        throw "Required OpenSSH executable was not found: $executable"
    }
}

$clientKeyDirectory = Join-Path $env:USERPROFILE '.ssh'
$clientKeyPath = Join-Path $clientKeyDirectory 'pomelo_orbit_ed25519'
$clientPublicKeyPath = Join-Path $clientKeyDirectory 'pomelo_orbit_ed25519_pub'
$defaultClientPublicKeyPath = "$clientKeyPath.pub"
New-Item -ItemType Directory -Path $clientKeyDirectory -Force | Out-Null

$privateKeyExists = Test-Path -LiteralPath $clientKeyPath -PathType Leaf
$publicKeyExists = Test-Path -LiteralPath $clientPublicKeyPath -PathType Leaf
if ($privateKeyExists -xor $publicKeyExists) {
    throw "The dedicated Pomelo Orbit key pair is incomplete. Both $clientKeyPath and $clientPublicKeyPath must exist, or neither file must exist."
}
if (-not $privateKeyExists -and (Test-Path -LiteralPath $defaultClientPublicKeyPath -PathType Leaf)) {
    throw "Unexpected default OpenSSH public key file exists: $defaultClientPublicKeyPath. Remove or rename it before creating the dedicated key pair."
}

if (-not $privateKeyExists) {
    Write-Host "Generating integration client key: $clientKeyPath"
    Invoke-NativeCommand $sshKeygen @(
        '-q', '-t', 'ed25519', '-N', '',
        '-C', 'pomelo-orbit-integration', '-f', $clientKeyPath
    )
}

Assert-UnencryptedOpenSSHPrivateKey $clientKeyPath
$derivedPublicKey = (& $sshKeygen -y -f $clientKeyPath).Trim()
if ($LASTEXITCODE -ne 0) {
    throw "Could not derive a public key from $clientKeyPath."
}
$derivedKeyIdentity = Get-PublicKeyIdentity $derivedPublicKey
if ($null -eq $derivedKeyIdentity) {
    throw "Could not derive a valid OpenSSH public key from $clientKeyPath."
}

if (-not $publicKeyExists) {
    [IO.File]::WriteAllText(
        $clientPublicKeyPath,
        ($derivedPublicKey + [Environment]::NewLine),
        [Text.Encoding]::ASCII
    )
    [IO.File]::Delete($defaultClientPublicKeyPath)
}

$authorizedPublicKey = (Get-Content -LiteralPath $clientPublicKeyPath -Raw).Trim()
$authorizedKeyIdentity = Get-PublicKeyIdentity $authorizedPublicKey
if ($null -eq $authorizedKeyIdentity) {
    throw "The dedicated public key is invalid: $clientPublicKeyPath"
}
if ($authorizedKeyIdentity -ne $derivedKeyIdentity) {
    throw "The dedicated key pair does not match: $clientKeyPath and $clientPublicKeyPath"
}

$programDataSsh = Join-Path $env:ProgramData 'ssh'
New-Item -ItemType Directory -Path $programDataSsh -Force | Out-Null
Set-OpenSSHRestrictedAcl -Path $programDataSsh -Directory
$openSshLogsDirectory = Join-Path $programDataSsh 'logs'
if (Test-Path -LiteralPath $openSshLogsDirectory -PathType Container) {
    Set-OpenSSHRestrictedAcl -Path $openSshLogsDirectory -Directory
}
$hostKeyPath = Join-Path $programDataSsh 'ssh_host_ed25519_key'
if ($RegenerateHostKey -and (Test-Path -LiteralPath $hostKeyPath)) {
    Remove-Item -LiteralPath $hostKeyPath -Force
    Remove-Item -LiteralPath "$hostKeyPath.pub" -Force -ErrorAction SilentlyContinue
}
if (-not (Test-Path -LiteralPath $hostKeyPath -PathType Leaf)) {
    Write-Host 'Generating Ed25519 SSH server host key...'
    Invoke-NativeCommand $sshKeygen @('-q', '-t', 'ed25519', '-N', '', '-f', $hostKeyPath)
}
if (-not (Test-Path -LiteralPath "$hostKeyPath.pub" -PathType Leaf)) {
    $hostPublicKey = & $sshKeygen -y -f $hostKeyPath
    if ($LASTEXITCODE -ne 0) { throw 'Failed to derive the server public host key.' }
    [IO.File]::WriteAllText(
        "$hostKeyPath.pub",
        ($hostPublicKey.Trim() + [Environment]::NewLine),
        [Text.Encoding]::ASCII
    )
}
$authorizedKeysPath = Join-Path $programDataSsh 'administrators_authorized_keys'
$authorizedKeys = @()
if (-not $ReplaceAuthorizedKeys -and (Test-Path -LiteralPath $authorizedKeysPath -PathType Leaf)) {
    $authorizedKeys = @(
        Get-Content -LiteralPath $authorizedKeysPath |
            ForEach-Object { $_.Trim() } |
            Where-Object { $_ -ne '' }
    )
}
$hasAuthorizedKey = $false
foreach ($existingKey in $authorizedKeys) {
    $existingKeyIdentity = Get-PublicKeyIdentity $existingKey
    if ($null -ne $existingKeyIdentity -and $existingKeyIdentity -eq $authorizedKeyIdentity) {
        $hasAuthorizedKey = $true
        break
    }
}
if (-not $hasAuthorizedKey) {
    $authorizedKeys += $authorizedPublicKey
}
[IO.File]::WriteAllLines($authorizedKeysPath, $authorizedKeys, [Text.Encoding]::ASCII)
Set-OpenSSHRestrictedAcl -Path $authorizedKeysPath

$configPath = Join-Path $programDataSsh 'sshd_config'
$configBackupPath = "$configPath.pomelo-orbit-backup"
if ((Test-Path -LiteralPath $configPath) -and -not (Test-Path -LiteralPath $configBackupPath)) {
    Copy-Item -LiteralPath $configPath -Destination $configBackupPath
}

$configLines = @(
    "Port $Port",
    "ListenAddress $ListenAddress",
    'HostKey __PROGRAMDATA__/ssh/ssh_host_ed25519_key',
    'PubkeyAuthentication yes',
    'AuthenticationMethods publickey',
    'PasswordAuthentication no',
    'KbdInteractiveAuthentication no',
    'PermitEmptyPasswords no',
    'AllowAgentForwarding no',
    'AllowTcpForwarding no',
    'GatewayPorts no',
    'PermitTunnel no',
    'X11Forwarding no',
    'Subsystem sftp sftp-server.exe',
    '',
    'Match Group administrators',
    '    AuthorizedKeysFile __PROGRAMDATA__/ssh/administrators_authorized_keys'
)
[IO.File]::WriteAllLines($configPath, $configLines, [Text.Encoding]::ASCII)
Set-OpenSSHRestrictedAcl -Path $configPath
Get-ChildItem -LiteralPath $programDataSsh -Filter 'ssh_host_*' -File |
    ForEach-Object { Set-OpenSSHRestrictedAcl -Path $_.FullName }
Invoke-NativeCommand $sshd @('-t', '-f', $configPath)

Write-Host 'Configuring Windows Firewall...'
$builtInFirewallRule = Get-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -ErrorAction SilentlyContinue
if ($null -ne $builtInFirewallRule) {
    $builtInFirewallRule | Disable-NetFirewallRule | Out-Null
}

$firewallGroup = 'Pomelo Orbit OpenSSH'
Get-NetFirewallRule -Group $firewallGroup -ErrorAction SilentlyContinue |
    Remove-NetFirewallRule -ErrorAction SilentlyContinue

if (-not $isLoopback) {
    $firewallLocalAddress = $ListenAddress
    if ($ListenAddress -eq '0.0.0.0' -or $ListenAddress -eq '::') {
        $firewallLocalAddress = 'Any'
    }
    New-NetFirewallRule `
        -Name "Pomelo-Orbit-OpenSSH-TCP-$Port" `
        -DisplayName "Pomelo Orbit OpenSSH ($Port)" `
        -Group $firewallGroup `
        -Enabled True `
        -Direction Inbound `
        -Action Allow `
        -Protocol TCP `
        -LocalAddress $firewallLocalAddress `
        -LocalPort $Port `
        -RemoteAddress $FirewallRemoteAddress `
        -Profile $FirewallProfile | Out-Null
}

New-Item -ItemType Directory -Path $WorkspacePath -Force | Out-Null

$service = Get-Service -Name sshd
Set-Service -Name sshd -StartupType Automatic
if ($service.Status -eq 'Running') {
    Restart-Service -Name sshd -Force
} else {
    Start-Service -Name sshd
}

$listener = $null
for ($attempt = 0; $attempt -lt 20; $attempt++) {
    $listener = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue |
        Where-Object { $_.LocalAddress -eq $ListenAddress } |
        Select-Object -First 1
    if ($null -ne $listener) { break }
    Start-Sleep -Milliseconds 250
}
if ($null -eq $listener) {
    throw "sshd is running but no listener was found on $ListenAddress`:$Port"
}

$fingerprintOutput = & $sshKeygen -lf "$hostKeyPath.pub" -E sha256
if ($LASTEXITCODE -ne 0) {
    throw 'Failed to calculate the SSH server host-key fingerprint.'
}
$hostKeyFingerprint = [regex]::Match(
    ($fingerprintOutput -join ' '),
    'SHA256:[A-Za-z0-9+/]+'
).Value
if ([string]::IsNullOrWhiteSpace($hostKeyFingerprint)) {
    throw "Failed to parse the SSH server host-key fingerprint: $fingerprintOutput"
}

$connectAddress = $ListenAddress
if ($ListenAddress -eq '0.0.0.0') { $connectAddress = '127.0.0.1' }
if ($ListenAddress -eq '::') { $connectAddress = '::1' }

$environmentHostValue = $EnvironmentHost
if ([string]::IsNullOrWhiteSpace($environmentHostValue)) {
    if ($ListenAddress -eq '0.0.0.0' -or $ListenAddress -eq '::') {
        $environmentHostValue = $env:COMPUTERNAME
    } else {
        $environmentHostValue = $ListenAddress
    }
}

$connectionTest = 'Skipped'
$canTestClientKey = $true

if (-not $SkipConnectionTest -and $canTestClientKey) {
    $sshUserCandidates = @(
        [Security.Principal.WindowsIdentity]::GetCurrent().Name,
        $env:USERNAME
    ) | Select-Object -Unique
    $connectionError = $null
    foreach ($sshUser in $sshUserCandidates) {
        $previousErrorActionPreference = $ErrorActionPreference
        $ErrorActionPreference = 'Continue'
        try {
            $output = & $ssh `
                -p $Port `
                -i $clientKeyPath `
                -o BatchMode=yes `
                -o ConnectTimeout=10 `
                -o StrictHostKeyChecking=no `
                -o UserKnownHostsFile=NUL `
                -o LogLevel=ERROR `
                -l $sshUser `
                $connectAddress `
                'whoami' 2>&1
            $sshExitCode = $LASTEXITCODE
        } finally {
            $ErrorActionPreference = $previousErrorActionPreference
        }
        if ($sshExitCode -eq 0) {
            $connectionTest = ($output -join ' ').Trim()
            $connectionError = $null
            break
        }
        $connectionError = ($output -join [Environment]::NewLine)
    }
    if ($null -ne $connectionError) {
        throw "SSH public-key connection test failed:`n$connectionError"
    }
}

$environmentUsername = [Security.Principal.WindowsIdentity]::GetCurrent().Name
$deploymentSSHPrivateKey = [IO.File]::ReadAllText($clientKeyPath, [Text.Encoding]::ASCII).Trim()

Write-Host ''
Write-Host 'Pomelo Orbit Windows SSH target is ready:'
[pscustomobject]@{
    Platform = 'windows'
    Host = $environmentHostValue
    Port = $Port
    Username = $environmentUsername
    HostKeyFingerprint = $hostKeyFingerprint
    WorkspaceRoot = $WorkspacePath
    PrivateKeyPath = $clientKeyPath
    SshConnectionTest = $connectionTest
    DockerPrerequisites = 'Verify with the Pomelo Orbit Environment Probe.'
} | Format-List

Write-Host 'Paste these values into Project > Edit deployment environment:'
[pscustomobject]@{
    State = 'active'
    Platform = 'windows'
    Host = $environmentHostValue
    Port = $Port
    Username = $environmentUsername
    WorkspaceRoot = $WorkspacePath
    HostKeyFingerprint = $hostKeyFingerprint
    DeploymentSSHPrivateKey = $deploymentSSHPrivateKey
    DeploymentSSHKeyPassphrase = ''
} | Format-List