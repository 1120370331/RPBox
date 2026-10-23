#requires -Version 7.0
param(
    [Parameter(Mandatory)][string]$TargetDirectory,
    [Parameter(Mandatory)][string]$ConfigPath
)

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$cli = Get-Content -LiteralPath (Join-Path $projectRoot 'client/node_modules/@tauri-apps/cli/package.json') -Raw | ConvertFrom-Json
if ($cli.version -notmatch '^\d+\.\d+\.\d+$') { throw 'Unsupported Tauri CLI version.' }
$toolsDirectory = Join-Path ([IO.Path]::GetFullPath($TargetDirectory)) '.tauri'
New-Item -ItemType Directory -Force -Path $toolsDirectory | Out-Null
$manifestPath = Join-Path $toolsDirectory "nsis-$($cli.version).json"
if (Test-Path -LiteralPath $manifestPath) {
    $manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
} else {
    # Read download URLs and checksums from the exact installed CLI release, not dev/latest.
    $source = Invoke-RestMethod "https://raw.githubusercontent.com/tauri-apps/tauri/tauri-cli-v$($cli.version)/crates/tauri-bundler/src/bundle/windows/nsis/mod.rs"
    $manifest = @{}
    foreach ($name in @('NSIS_URL', 'NSIS_SHA1', 'NSIS_TAURI_UTILS_URL', 'NSIS_TAURI_UTILS_SHA1')) {
        $match = [regex]::Match($source, "const ${name}: &str =\s*`"([^`"]+)`";")
        if (-not $match.Success) { throw "Cannot resolve $name for Tauri $($cli.version)." }
        $manifest[$name] = $match.Groups[1].Value
    }
    $manifest | ConvertTo-Json | Set-Content -LiteralPath $manifestPath -Encoding utf8NoBOM
}

function Get-VerifiedTool([string]$Url, [string]$Hash, [string]$Destination) {
    if ($Url -notmatch '^https://github\.com/tauri-apps/' -or $Hash -notmatch '^[A-Fa-f0-9]{40}$') {
        throw 'Invalid Tauri tool download metadata.'
    }
    if ((Test-Path -LiteralPath $Destination) -and (Get-FileHash -LiteralPath $Destination -Algorithm SHA1).Hash -eq $Hash) { return }
    Invoke-WebRequest -Uri $Url -OutFile "$Destination.part"
    if ((Get-FileHash -LiteralPath "$Destination.part" -Algorithm SHA1).Hash -ne $Hash) { throw 'Tauri tool checksum mismatch.' }
    Move-Item -LiteralPath "$Destination.part" -Destination $Destination -Force
}

$nsisDirectory = Join-Path $toolsDirectory 'NSIS'
if (-not (Test-Path -LiteralPath (Join-Path $nsisDirectory 'Bin/makensis.exe'))) {
    $zipPath = Join-Path $toolsDirectory 'nsis.zip'
    Get-VerifiedTool $manifest.NSIS_URL $manifest.NSIS_SHA1 $zipPath
    # The verified official archive contains a single versioned nsis-* root directory.
    $archive = [IO.Compression.ZipFile]::OpenRead($zipPath)
    try { $rootName = ($archive.Entries[0].FullName -split '/')[0] } finally { $archive.Dispose() }
    if ($rootName -notmatch '^nsis-[0-9.]+$') { throw 'Unexpected NSIS archive layout.' }
    Expand-Archive -LiteralPath $zipPath -DestinationPath $toolsDirectory -Force
    New-Item -ItemType Directory -Force -Path $nsisDirectory | Out-Null
    Get-ChildItem -LiteralPath (Join-Path $toolsDirectory $rootName) -Force |
        Copy-Item -Destination $nsisDirectory -Recurse -Force
}
$pluginDirectory = Join-Path $nsisDirectory 'Plugins/x86-unicode/additional'
New-Item -ItemType Directory -Force -Path $pluginDirectory | Out-Null
Get-VerifiedTool $manifest.NSIS_TAURI_UTILS_URL $manifest.NSIS_TAURI_UTILS_SHA1 (Join-Path $pluginDirectory 'nsis_tauri_utils.dll')
@{ bundle = @{ useLocalToolsDir = $true } } | ConvertTo-Json | Set-Content -LiteralPath $ConfigPath -Encoding utf8NoBOM
Write-Host "NSIS cache ready for Tauri $($cli.version): $toolsDirectory"
