#requires -Version 7.0
# Exercise the real metadata and promotion scripts with local transport adapters.
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$bash = 'C:/Program Files/Git/bin/bash.exe'
if (-not (Test-Path -LiteralPath $bash)) { throw 'Git Bash is required for the local remote-command test.' }
$testRoot = Join-Path ([IO.Path]::GetTempPath()) ('rpbox-release-test-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $testRoot | Out-Null
$remoteWindows = Join-Path $testRoot 'remote'
$remote = '/' + $remoteWindows.Substring(0, 1).ToLowerInvariant() + $remoteWindows.Substring(2).Replace('\', '/')
$bundle = Join-Path $testRoot 'bundle'
$artifacts = Join-Path $testRoot 'artifacts'
New-Item -ItemType Directory -Path $bundle, $remoteWindows | Out-Null
$env:SSH_PRIVATE_KEY = 'test-only-key'
$env:SSH_KNOWN_HOSTS = 'test-only-host'
$global:RPBoxReleaseTestState = @{ failUpload = $false; corruptUpload = $false; connections = 0 }

function ssh {
    $global:RPBoxReleaseTestState.connections++
    $command = $args[-1]
    & $bash -c $command
    $global:LASTEXITCODE = $LASTEXITCODE
}
function scp {
    $global:RPBoxReleaseTestState.connections++
    if ($global:RPBoxReleaseTestState.failUpload) { $global:LASTEXITCODE = 1; return }
    $destination = ($args[-1] -split ':', 2)[1]
    $localDestination = $destination.Substring(1, 1) + ':' + $destination.Substring(2).Replace('/', '\')
    # The publisher passes exactly four artifact files before the destination.
    foreach ($source in $args[($args.Count - 5)..($args.Count - 2)]) {
        Copy-Item -LiteralPath $source -Destination $localDestination
    }
    if ($global:RPBoxReleaseTestState.corruptUpload) {
        Add-Content -LiteralPath (Join-Path $localDestination 'latest.json') -Value 'corrupt'
    }
    $global:LASTEXITCODE = 0
}
function Assert-True([bool]$Value, [string]$Message) {
    if (-not $Value) { throw $Message }
}
function Assert-Fails([scriptblock]$Action, [string]$Message) {
    $failed = $false
    try { & $Action } catch { $failed = $true }
    Assert-True $failed $Message
}
function Prepare([string]$Version, [string]$Channel) {
    Set-Content -LiteralPath (Join-Path $bundle "RPBox_${Version}_x64-setup.exe") 'installer fixture'
    Set-Content -LiteralPath (Join-Path $bundle "RPBox_${Version}_x64-setup.exe.sig") 'signature fixture'
    & "$projectRoot/scripts/prepare-client-release.ps1" -Version $Version -Channel $Channel -BundleDirectory $bundle -OutputDirectory $artifacts
}
function Publish([string]$Version, [string]$Channel) {
    & "$projectRoot/scripts/publish-client-release.ps1" -Version $Version -Channel $Channel -ArtifactDirectory $artifacts -ServerHost localhost -ServerUser fixture -RemotePath $remote -Attempts 1
}
try {
    Prepare '7.8.9' stable
    Publish '7.8.9' stable
    $stablePath = Join-Path $remoteWindows 'latest.json'
    Assert-True ((Get-Content $stablePath -Raw | ConvertFrom-Json).latest_version -eq '7.8.9') 'Stable release did not publish.'
    Assert-True ($global:RPBoxReleaseTestState.connections -eq 4) 'Expected mkdir + one SCP + promotion + cleanup.'
    $stableHash = (Get-FileHash $stablePath).Hash
    Prepare '7.8.10' stable
    & "$projectRoot/scripts/publish-client-release.ps1" -Version '7.8.10' -Channel stable -ArtifactDirectory $artifacts -ServerHost localhost -ServerUser fixture -RemotePath $remote -Attempts 1 -VerifyOnly
    Assert-True ((Get-FileHash $stablePath).Hash -eq $stableHash) 'VerifyOnly changed the stable pointer.'
    Assert-True (-not (Test-Path (Join-Path $remoteWindows '7.8.10'))) 'VerifyOnly created a published version directory.'
    Prepare '7.8.10-1' beta
    Publish '7.8.10-1' beta
    Assert-True ((Get-Content (Join-Path $remoteWindows 'latest-beta.json') -Raw | ConvertFrom-Json).latest_version -eq '7.8.10-1') 'Beta release did not publish.'
    Assert-True ((Get-FileHash $stablePath).Hash -eq $stableHash) 'Beta publish changed the stable pointer.'
    Prepare '7.8.11' stable
    $global:RPBoxReleaseTestState.failUpload = $true
    Assert-Fails { Publish '7.8.11' stable } 'Failed SCP was accepted.'
    Assert-True ((Get-FileHash $stablePath).Hash -eq $stableHash) 'Failed upload changed the stable pointer.'
    $global:RPBoxReleaseTestState.failUpload = $false
    $global:RPBoxReleaseTestState.corruptUpload = $true
    Assert-Fails { Publish '7.8.11' stable } 'Corrupt remote file was accepted.'
    Assert-True ((Get-FileHash $stablePath).Hash -eq $stableHash) 'Corrupt upload changed the stable pointer.'
    $global:RPBoxReleaseTestState.corruptUpload = $false
    $before = $global:RPBoxReleaseTestState.connections
    Assert-Fails { Publish '7.8.11' beta } 'Mismatched channel was accepted.'
    Add-Content -LiteralPath (Join-Path $artifacts 'RPBox_7.8.11_x64-setup.exe') 'changed'
    Assert-Fails { Publish '7.8.11' stable } 'Local checksum mismatch was accepted.'
    Assert-True ($global:RPBoxReleaseTestState.connections -eq $before) 'Invalid local artifacts opened a connection.'
    Set-Content -LiteralPath (Join-Path $bundle 'RPBox_7.8.11_x64-setup.exe.sig') ' '
    Assert-Fails {
        & "$projectRoot/scripts/prepare-client-release.ps1" -Version '7.8.11' -BundleDirectory $bundle -OutputDirectory $artifacts
    } 'Blank signature was accepted.'
    Write-Host 'PASS: stable/beta publishing, upload/checksum failures, channel mismatch, blank signature.'
} finally {
    Remove-Item Env:SSH_PRIVATE_KEY, Env:SSH_KNOWN_HOSTS -ErrorAction SilentlyContinue
    $resolved = [IO.Path]::GetFullPath($testRoot)
    if ([IO.Path]::GetDirectoryName($resolved) -eq [IO.Path]::GetTempPath().TrimEnd('\', '/') -and
        [IO.Path]::GetFileName($resolved) -like 'rpbox-release-test-*') {
        Remove-Item -LiteralPath $resolved -Recurse -Force
    }
}
