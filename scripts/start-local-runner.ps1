#requires -Version 7.0
param(
    [ValidatePattern('^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$')][string]$Repository = '1120370331/RPBox',
    [string]$RunnerDirectory = (Join-Path $env:LOCALAPPDATA 'RPBox/actions-runner'),
    [switch]$InstallOnly
)

$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $false
if (-not $IsWindows -or [Runtime.InteropServices.RuntimeInformation]::OSArchitecture -ne 'X64') {
    throw 'This runner requires Windows x64.'
}
foreach ($command in @('gh', 'git', 'node', 'npm', 'cargo', 'rustup', 'pwsh', 'ssh', 'scp', 'ssh-keyscan')) {
    Get-Command $command -ErrorAction Stop | Out-Null
}
$vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
if (-not (Test-Path -LiteralPath $vswhere)) { throw 'Install Visual Studio C++ Build Tools first.' }
$vsPath = & $vswhere -latest -products '*' -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
if (-not $vsPath) { throw 'Install the Visual Studio Desktop development with C++ workload first.' }
& rustup run stable cargo --version
if ($LASTEXITCODE -ne 0) { throw 'Install the stable MSVC Rust toolchain first: rustup toolchain install stable' }

$runnerRoot = [IO.Path]::GetFullPath($RunnerDirectory).TrimEnd('\', '/')
$repositoryRoot = [IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot)).TrimEnd('\', '/')
$commonGitDirectory = & git -C $repositoryRoot rev-parse --path-format=absolute --git-common-dir
if ($LASTEXITCODE -ne 0) { throw 'Start this script from an RPBox Git checkout.' }
$localObjects = Join-Path $commonGitDirectory 'objects'
if ($runnerRoot -eq $repositoryRoot -or $runnerRoot.StartsWith($repositoryRoot + '\', [StringComparison]::OrdinalIgnoreCase)) {
    throw 'RunnerDirectory must be outside the development checkout.'
}
if (Test-Path -LiteralPath (Join-Path $runnerRoot '.runner')) {
    throw "This directory already contains a registered runner. Finish or remove it with config.cmd remove before restarting: $runnerRoot"
}
New-Item -ItemType Directory -Force -Path $runnerRoot | Out-Null
# Prevent two launchers from configuring/updating the same installation.
$lock = [IO.File]::Open((Join-Path $runnerRoot '.launcher.lock'), 'OpenOrCreate', 'ReadWrite', 'None')
try {
    $packagesJson = & gh api "repos/$Repository/actions/runners/downloads"
    if ($LASTEXITCODE -ne 0) { throw 'Cannot read runner packages. Log in with gh using repository administration access.' }
    $package = @($packagesJson | ConvertFrom-Json | Where-Object { $_.os -eq 'win' -and $_.architecture -eq 'x64' })[0]
    if (-not $package -or $package.sha256_checksum -notmatch '^[a-fA-F0-9]{64}$') {
        throw 'GitHub did not return a Windows x64 runner with a SHA256 checksum.'
    }
    $marker = Join-Path $runnerRoot '.rpbox-package'
    if (-not (Test-Path -LiteralPath $marker) -or (Get-Content -LiteralPath $marker -Raw).Trim() -ne $package.sha256_checksum -or
        -not (Test-Path -LiteralPath (Join-Path $runnerRoot 'bin/Runner.Listener.exe'))) {
        $archive = Join-Path $runnerRoot 'runner-download.zip'
        Write-Host "Downloading $($package.filename)..."
        Invoke-WebRequest -Uri $package.download_url -OutFile $archive
        if ((Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash -ne $package.sha256_checksum) {
            throw 'Runner download checksum mismatch.'
        }
        Expand-Archive -LiteralPath $archive -DestinationPath $runnerRoot -Force
        Remove-Item -LiteralPath $archive
        $package.sha256_checksum | Set-Content -LiteralPath $marker -Encoding utf8NoBOM
    }
    Write-Host "Runner installed: $runnerRoot"
    if ($InstallOnly) { return }

    Push-Location $runnerRoot
    try {
        $registrationJson = & gh api --method POST "repos/$Repository/actions/runners/registration-token"
        if ($LASTEXITCODE -ne 0) { throw 'Cannot register a runner. Repository administration access is required.' }
        # The supported environment input avoids exposing tokens in process arguments.
        $env:ACTIONS_RUNNER_INPUT_TOKEN = ($registrationJson | ConvertFrom-Json).token
        try {
            & ./config.cmd --unattended --url "https://github.com/$Repository" --name "rpbox-$env:COMPUTERNAME-$PID" --labels rpbox-release --work _work --ephemeral
            if ($LASTEXITCODE -ne 0) { throw 'Runner registration failed.' }
        } finally {
            Remove-Item Env:ACTIONS_RUNNER_INPUT_TOKEN -ErrorAction SilentlyContinue
            $registrationJson = $null
        }
        Write-Host 'Ready for one trusted Release Client job with builder=local; the runner exits after that job.'
        # Do not inherit developer URL rewrites, credential helpers or GitHub tokens.
        $savedEnvironment = @{}
        foreach ($name in @('GH_TOKEN', 'GITHUB_TOKEN', 'GH_ENTERPRISE_TOKEN', 'GITHUB_ENTERPRISE_TOKEN',
            'GIT_CONFIG_GLOBAL', 'GIT_CONFIG_NOSYSTEM', 'GIT_TERMINAL_PROMPT', 'GIT_ALTERNATE_OBJECT_DIRECTORIES')) {
            $savedEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
            [Environment]::SetEnvironmentVariable($name, $null, 'Process')
        }
        $env:GIT_CONFIG_GLOBAL = Join-Path $runnerRoot 'ci.gitconfig'
        "[core]`n    autocrlf = false`n    longpaths = true`n" | Set-Content -LiteralPath $env:GIT_CONFIG_GLOBAL -Encoding utf8NoBOM
        $env:GIT_CONFIG_NOSYSTEM = '1'
        $env:GIT_TERMINAL_PROMPT = '0'
        # Read committed objects locally, while checkout still verifies the exact remote SHA.
        # This does not copy working-tree files or change the development repository.
        $env:GIT_ALTERNATE_OBJECT_DIRECTORIES = $localObjects.Replace('\', '/')
        try {
            & ./run.cmd
            if ($LASTEXITCODE -ne 0) { throw "Runner exited with code $LASTEXITCODE." }
        } finally {
            foreach ($name in $savedEnvironment.Keys) { [Environment]::SetEnvironmentVariable($name, $savedEnvironment[$name], 'Process') }
        }
    } finally {
        # Also unregister after Ctrl+C or a failed job, retaining only build caches.
        if (Test-Path -LiteralPath (Join-Path $runnerRoot '.runner')) {
            $removeJson = & gh api --method POST "repos/$Repository/actions/runners/remove-token"
            if ($LASTEXITCODE -eq 0) {
                $env:ACTIONS_RUNNER_INPUT_TOKEN = ($removeJson | ConvertFrom-Json).token
                try { & ./config.cmd remove --unattended } finally {
                    Remove-Item Env:ACTIONS_RUNNER_INPUT_TOKEN -ErrorAction SilentlyContinue
                }
            } else { Write-Warning "Runner cleanup failed. Run config.cmd remove in $runnerRoot before restarting." }
        }
        Pop-Location
    }
} finally {
    $lock.Dispose()
}
