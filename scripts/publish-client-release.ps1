#requires -Version 7.0
param(
    [Parameter(Mandatory)][ValidatePattern('^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$')][string]$Version,
    [ValidateSet('stable', 'beta')][string]$Channel = 'stable',
    [Parameter(Mandatory)][string]$ArtifactDirectory,
    [Parameter(Mandatory)][ValidatePattern('^[A-Za-z0-9][A-Za-z0-9.-]*$')][string]$ServerHost,
    [Parameter(Mandatory)][ValidatePattern('^[A-Za-z_][A-Za-z0-9_-]*$')][string]$ServerUser,
    [Parameter(Mandatory)][ValidatePattern('^/[A-Za-z0-9._/-]+$')][string]$RemotePath,
    [ValidateRange(1, 65535)][int]$ServerPort = 2233,
    [ValidateRange(1, 3)][int]$Attempts = 3,
    [switch]$VerifyOnly
)

$ErrorActionPreference = 'Stop'
# Native failures are retried explicitly, never ignored by PowerShell.
$PSNativeCommandUseErrorActionPreference = $false
if ($RemotePath -match '(^|/)\.\.(/|$)' -or $RemotePath.Trim('/') -eq '') {
    throw 'RemotePath must be an absolute release directory without parent traversal.'
}
if (-not $env:SSH_PRIVATE_KEY) { throw 'SSH_PRIVATE_KEY is missing.' }
$fileName = "RPBox_${Version}_x64-setup.exe"
$names = @($fileName, "$fileName.sig", 'latest.json', 'SHA256SUMS')
foreach ($name in $names) {
    $file = Join-Path $ArtifactDirectory $name
    if (-not (Test-Path -LiteralPath $file -PathType Leaf) -or (Get-Item -LiteralPath $file).Length -eq 0) {
        throw "Missing or empty release artifact: $name"
    }
}
$metadata = Get-Content -LiteralPath (Join-Path $ArtifactDirectory 'latest.json') -Raw | ConvertFrom-Json
if ($metadata.latest_version -ne $Version -or $metadata.channel -ne $Channel) {
    throw 'Release metadata does not match the requested version/channel.'
}
# Verify locally before opening a connection; verify again on the server before promotion.
$expectedSums = foreach ($name in $names[0..2]) {
    $hash = (Get-FileHash -LiteralPath (Join-Path $ArtifactDirectory $name) -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $name"
}
$actualSums = @(Get-Content -LiteralPath (Join-Path $ArtifactDirectory 'SHA256SUMS'))
if (@(Compare-Object $expectedSums $actualSums).Count -ne 0) { throw 'Release checksum mismatch.' }

function Invoke-TransferCommand {
    param([string]$Program, [string[]]$Arguments)
    for ($attempt = 1; $attempt -le $Attempts; $attempt++) {
        & $Program @Arguments
        if ($LASTEXITCODE -eq 0) { return }
        if ($attempt -eq $Attempts) { throw "$Program failed after $Attempts attempt(s)." }
        Start-Sleep -Seconds (5 * $attempt)
    }
}

$tempRoot = [IO.Path]::GetTempPath()
$tempDirectory = Join-Path $tempRoot ("rpbox-publish-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tempDirectory | Out-Null
$keyPath = Join-Path $tempDirectory 'deploy.key'
$knownHosts = Join-Path $tempDirectory 'known_hosts'
try {
    if ($IsWindows) {
        # Restrict the directory before putting the private key in it.
        $sid = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
        & icacls $tempDirectory /inheritance:r /grant:r "*${sid}:(OI)(CI)F" | Out-Null
        if ($LASTEXITCODE -ne 0) { throw 'Failed to restrict deployment key permissions.' }
    } else {
        & chmod 700 $tempDirectory
        if ($LASTEXITCODE -ne 0) { throw 'Failed to restrict deployment directory permissions.' }
    }
    ($env:SSH_PRIVATE_KEY.Trim().Replace("`r", '') + "`n") | Set-Content -LiteralPath $keyPath -NoNewline -Encoding utf8NoBOM
    if (-not $IsWindows) {
        & chmod 600 $keyPath
        if ($LASTEXITCODE -ne 0) { throw 'Failed to restrict deployment key permissions.' }
    }
    # Optional pinned host key; otherwise preserve the existing workflow's keyscan behavior.
    $hostKeys = if ($env:SSH_KNOWN_HOSTS) { $env:SSH_KNOWN_HOSTS } else {
        $scanned = & ssh-keyscan -T 15 -p $ServerPort $ServerHost 2>$null
        if ($LASTEXITCODE -ne 0 -or -not $scanned) { throw 'Could not obtain the server SSH host key.' }
        $scanned -join "`n"
    }
    $hostKeys | Set-Content -LiteralPath $knownHosts -Encoding utf8NoBOM
    $common = @('-F', 'none', '-i', $keyPath, '-o', 'IdentitiesOnly=yes', '-o', 'BatchMode=yes',
        '-o', 'StrictHostKeyChecking=yes', '-o', "UserKnownHostsFile=`"$knownHosts`"",
        '-o', 'ConnectTimeout=30', '-o', 'ServerAliveInterval=15', '-o', 'ServerAliveCountMax=4')
    $target = "$ServerUser@$ServerHost"
    $remoteBase = $RemotePath.TrimEnd('/')
    $remoteDirectory = "$remoteBase/$Version"
    $staging = "$remoteBase/.upload-$Version-$([guid]::NewGuid().ToString('N'))"
    $metadataFile = if ($Channel -eq 'beta') { 'latest-beta.json' } else { 'latest.json' }
    Invoke-TransferCommand ssh (@('-p', "$ServerPort") + $common + @($target, "mkdir -p '$staging'"))
    # One SCP connection for all four files; installers are already compressed.
    $files = @($names | ForEach-Object { Join-Path $ArtifactDirectory $_ })
    Invoke-TransferCommand scp (@('-P', "$ServerPort") + $common + $files + @("${target}:$staging/"))
    # Publish the channel pointer last. A failed upload/checksum leaves the old channel intact.
    # Retry-safe: keep staging sources until the remote command has succeeded.
    $promote = @"
set -eu
cd '$staging'
sha256sum -c SHA256SUMS
mkdir -p '$remoteDirectory'
cp '$fileName' '$remoteDirectory/$fileName.part'
cp '$fileName.sig' '$remoteDirectory/$fileName.sig.part'
cp latest.json '$remoteDirectory/latest.json.part'
cp SHA256SUMS '$remoteDirectory/SHA256SUMS.part'
mv -f '$remoteDirectory/$fileName.part' '$remoteDirectory/$fileName'
mv -f '$remoteDirectory/$fileName.sig.part' '$remoteDirectory/$fileName.sig'
mv -f '$remoteDirectory/latest.json.part' '$remoteDirectory/latest.json'
mv -f '$remoteDirectory/SHA256SUMS.part' '$remoteDirectory/SHA256SUMS'
cp latest.json '$staging/$metadataFile.ready'
mv -f '$staging/$metadataFile.ready' '$remoteBase/$metadataFile'
"@
    if ($VerifyOnly) { $promote = "set -eu`ncd '$staging'`nsha256sum -c SHA256SUMS" }
    Invoke-TransferCommand ssh (@('-p', "$ServerPort") + $common + @($target, $promote.Replace("`r", '')))
    # Cleanup failure must not turn a successfully published release into a failed release.
    & ssh -p $ServerPort @common $target "rm -f '$staging/$fileName' '$staging/$fileName.sig' '$staging/latest.json' '$staging/SHA256SUMS' && rmdir '$staging'"
    if ($LASTEXITCODE -ne 0) { Write-Warning "Published successfully; staging cleanup remains: $staging" }
    if ($VerifyOnly) { Write-Host "Verified upload for $Channel client $Version; no update published." }
    else { Write-Host "Published $Channel client $Version." }
} finally {
    # Only the unique directory created by this invocation may be removed.
    $resolved = [IO.Path]::GetFullPath($tempDirectory)
    if ([IO.Path]::GetDirectoryName($resolved) -eq [IO.Path]::GetFullPath($tempRoot).TrimEnd('\', '/') -and
        [IO.Path]::GetFileName($resolved) -like 'rpbox-publish-*') {
        Remove-Item -LiteralPath $resolved -Recurse -Force
    }
}
