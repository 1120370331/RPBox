#requires -Version 7.0
param(
    [Parameter(Mandatory)][ValidatePattern('^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$')][string]$Version,
    [ValidateSet('stable', 'beta')][string]$Channel = 'stable',
    [Parameter(Mandatory)][string]$BundleDirectory,
    [Parameter(Mandatory)][string]$OutputDirectory,
    [string]$TagName = ''
)

$ErrorActionPreference = 'Stop'
if ($Channel -eq 'beta' -and $Version -notmatch '^\d+\.\d+\.\d+-\d+$') {
    throw 'Beta releases require a numeric prerelease, for example 1.0.7-1.'
}
$fileName = "RPBox_${Version}_x64-setup.exe"
$installer = Join-Path $BundleDirectory $fileName
$signature = "$installer.sig"
foreach ($file in @($installer, $signature)) {
    if (-not (Test-Path -LiteralPath $file -PathType Leaf) -or (Get-Item -LiteralPath $file).Length -eq 0) {
        throw "Missing or empty release artifact: $file"
    }
}
if ([string]::IsNullOrWhiteSpace((Get-Content -LiteralPath $signature -Raw))) {
    throw 'The updater signature is empty.'
}

$projectRoot = Split-Path -Parent $PSScriptRoot
$notesFile = Join-Path $projectRoot "client/release-notes/$Version.txt"
$betaNotesFile = Join-Path $projectRoot "client/release-notes/beta/$Version.txt"
if ($Channel -eq 'beta' -and (Test-Path -LiteralPath $betaNotesFile)) { $notesFile = $betaNotesFile }
$notes = if (Test-Path -LiteralPath $notesFile) { (Get-Content -LiteralPath $notesFile -Raw).Trim() } else { '' }
if (-not $notes -and $TagName) {
    $tagNotes = & git -C $projectRoot for-each-ref "refs/tags/$TagName" '--format=%(contents)'
    if ($LASTEXITCODE -ne 0) { throw 'Failed to read release tag notes.' }
    $notes = ($tagNotes -join "`n").Trim()
}
if (-not $notes) { $notes = '修复若干已知问题，优化使用体验并提升稳定性。' }

New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
Copy-Item -LiteralPath $installer, $signature -Destination $OutputDirectory -Force
$metadata = [ordered]@{
    latest_version = $Version
    notes = $notes.Replace("`r`n", "`n").Replace("`r", "`n")
    pub_date = [DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')
    channel = $Channel
}
$metadata | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $OutputDirectory 'latest.json') -Encoding utf8NoBOM
$checksums = foreach ($name in @($fileName, "$fileName.sig", 'latest.json')) {
    $hash = (Get-FileHash -LiteralPath (Join-Path $OutputDirectory $name) -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $name"
}
# GNU sha256sum treats CR as part of the filename; always emit LF on Windows too.
[IO.File]::WriteAllText((Join-Path $OutputDirectory 'SHA256SUMS'), ($checksums -join "`n") + "`n", [Text.UTF8Encoding]::new($false))
Write-Host "Prepared $Channel release $Version in $OutputDirectory"
