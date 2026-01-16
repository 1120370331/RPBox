# RPBox 插件发布脚本
# 用法: .\scripts\release-addon.ps1 -Version "1.1.0" [-Changelog "更新说明"]

param(
    [Parameter(Mandatory=$true)]
    [string]$Version,
    [string]$Changelog = "",
    [string]$SSHHost = "your-server.com",
    [string]$SSHUser = "root",
    [string]$RemotePath = "/var/www/rpbox/storage/addons/RPBox_Addon"
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$AddonDir = Join-Path $ProjectRoot "addon\RPBox_Addon"
$OutputDir = Join-Path $ProjectRoot "releases\addon\$Version"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  RPBox 插件发布脚本 v$Version" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# 1. 更新 TOC 版本号
Write-Host "`n[1/4] 更新版本号..." -ForegroundColor Yellow
$TocFile = Join-Path $AddonDir "RPBox_Addon.toc"
$TocContent = Get-Content $TocFile -Raw
$TocContent = $TocContent -replace '## Version: .*', "## Version: $Version"
Set-Content $TocFile $TocContent -Encoding UTF8
Write-Host "  RPBox_Addon.toc: 已更新到 $Version" -ForegroundColor Green

# 2. 打包插件
Write-Host "`n[2/4] 打包插件..." -ForegroundColor Yellow
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
$ZipFile = Join-Path $OutputDir "RPBox_Addon_$Version.zip"

# 创建临时目录用于打包
$TempDir = Join-Path $env:TEMP "RPBox_Addon_Pack"
if (Test-Path $TempDir) { Remove-Item $TempDir -Recurse -Force }
Copy-Item $AddonDir $TempDir -Recurse

# 压缩
Compress-Archive -Path $TempDir -DestinationPath $ZipFile -Force
Remove-Item $TempDir -Recurse -Force
Write-Host "  已生成: $ZipFile" -ForegroundColor Green

# 3. 上传到服务器
Write-Host "`n[3/4] 上传到服务器..." -ForegroundColor Yellow
ssh "${SSHUser}@${SSHHost}" "mkdir -p $RemotePath/$Version"
scp $ZipFile "${SSHUser}@${SSHHost}:${RemotePath}/${Version}/"

# 复制文件到 latest 目录
ssh "${SSHUser}@${SSHHost}" "rm -rf $RemotePath/latest && mkdir -p $RemotePath/latest"
$AddonFiles = Get-ChildItem -Path $AddonDir -File
foreach ($file in $AddonFiles) {
    scp $file.FullName "${SSHUser}@${SSHHost}:${RemotePath}/latest/"
}
# 复制 Locales 目录
scp -r (Join-Path $AddonDir "Locales") "${SSHUser}@${SSHHost}:${RemotePath}/latest/"

# 更新 latest.zip
ssh "${SSHUser}@${SSHHost}" "cp $RemotePath/$Version/RPBox_Addon_$Version.zip $RemotePath/latest.zip"
Write-Host "  上传完成" -ForegroundColor Green

# 4. 更新 manifest.json
Write-Host "`n[4/4] 更新 manifest.json..." -ForegroundColor Yellow
$ManifestUpdate = @"
{
  "name": "RPBox_Addon",
  "latest": "$Version",
  "versions": [
    {
      "version": "$Version",
      "releaseDate": "$(Get-Date -Format 'yyyy-MM-dd')",
      "minClientVersion": "1.0.0",
      "changelog": "$Changelog",
      "downloadUrl": "/api/v1/addon/download/$Version"
    }
  ]
}
"@

$ManifestFile = Join-Path $OutputDir "manifest.json"
Set-Content $ManifestFile $ManifestUpdate -Encoding UTF8
scp $ManifestFile "${SSHUser}@${SSHHost}:${RemotePath}/"
Write-Host "  manifest.json 已更新" -ForegroundColor Green

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "  插件发布完成!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
