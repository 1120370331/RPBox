# deploy-client.ps1 - 本地构建并部署客户端
# 用法: .\scripts\deploy-client.ps1 -Version "0.2.8"

param(
    [Parameter(Mandatory=$true)]
    [string]$Version,

    [string]$ServerHost = "your-server-host",
    [string]$ServerUser = "devbox",
    [int]$ServerPort = 2233,
    [string]$RemotePath = "/home/devbox/RPBox/server/releases",
    [string]$ApiBase = "https://ksxvodevhonx.sealosbja.site/api/v1",
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
if (-not $ProjectRoot) { $ProjectRoot = (Get-Location).Path }

Write-Host "=== RPBox Client Deploy Script ===" -ForegroundColor Cyan
Write-Host "Version: $Version"

# 1. 更新版本号
Write-Host "`n[1/5] Updating version numbers..." -ForegroundColor Yellow
$files = @(
    @{ Path = "client/src-tauri/tauri.conf.json"; Pattern = '"version": "[^"]*"'; Replace = "`"version`": `"$Version`"" },
    @{ Path = "client/src-tauri/Cargo.toml"; Pattern = 'version = "[^"]*"'; Replace = "version = `"$Version`"" },
    @{ Path = "client/package.json"; Pattern = '"version": "[^"]*"'; Replace = "`"version`": `"$Version`"" }
)

foreach ($file in $files) {
    $fullPath = Join-Path $ProjectRoot $file.Path
    $content = Get-Content $fullPath -Raw
    $content = $content -replace $file.Pattern, $file.Replace
    Set-Content $fullPath $content -NoNewline
    Write-Host "  Updated: $($file.Path)"
}

# 2. 构建
if (-not $SkipBuild) {
    Write-Host "`n[2/5] Building Tauri app..." -ForegroundColor Yellow
    Push-Location (Join-Path $ProjectRoot "client")
    try {
        $env:VITE_API_BASE = $ApiBase
        npm run tauri build
        if ($LASTEXITCODE -ne 0) { throw "Build failed" }
    } finally {
        Pop-Location
    }
} else {
    Write-Host "`n[2/5] Skipping build (--SkipBuild)" -ForegroundColor Gray
}

# 3. 查找构建产物
Write-Host "`n[3/5] Locating build artifacts..." -ForegroundColor Yellow
$bundlePath = Join-Path $ProjectRoot "client/src-tauri/target/release/bundle"
$exePath = Get-ChildItem -Path "$bundlePath/nsis" -Filter "RPBox_*_x64-setup.exe" -Recurse | Select-Object -First 1
$sigPath = "$($exePath.FullName).sig"

if (-not $exePath -or -not (Test-Path $sigPath)) {
    throw "Build artifacts not found. EXE: $exePath, SIG exists: $(Test-Path $sigPath)"
}
Write-Host "  EXE: $($exePath.Name)"
Write-Host "  SIG: $(Split-Path $sigPath -Leaf)"

# 4. 准备 latest.json
Write-Host "`n[4/5] Preparing metadata..." -ForegroundColor Yellow
$notesFile = Join-Path $ProjectRoot "client/release-notes/$Version.txt"
$notes = ""
if (Test-Path $notesFile) {
    $notes = Get-Content $notesFile -Raw
}
$pubDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$metadata = @{
    latest_version = $Version
    notes = $notes
    pub_date = $pubDate
} | ConvertTo-Json -Compress

$latestJsonPath = Join-Path $ProjectRoot "latest.json"
Set-Content $latestJsonPath $metadata -Encoding UTF8
Write-Host "  Created: latest.json"

# 5. 上传到服务器
Write-Host "`n[5/5] Uploading to server..." -ForegroundColor Yellow
$remoteDir = "$RemotePath/$Version"

# 创建远程目录
ssh -p $ServerPort "$ServerUser@$ServerHost" "mkdir -p $remoteDir"

# 上传文件（使用压缩）
scp -P $ServerPort -C $exePath.FullName "$ServerUser@$ServerHost`:$remoteDir/"
scp -P $ServerPort -C $sigPath "$ServerUser@$ServerHost`:$remoteDir/"
scp -P $ServerPort $latestJsonPath "$ServerUser@$ServerHost`:$RemotePath/latest.json"
scp -P $ServerPort $latestJsonPath "$ServerUser@$ServerHost`:$remoteDir/latest.json"

# 清理临时文件
Remove-Item $latestJsonPath -ErrorAction SilentlyContinue

Write-Host "`n=== Deploy Complete ===" -ForegroundColor Green
Write-Host "Version $Version deployed to $ServerHost"
Write-Host "Remember to update server config.yaml with new version info!"
