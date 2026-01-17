# RPBox 客户端发布脚本
# 用法: .\scripts\release.ps1 -Version "0.2.0" [-Notes "更新说明"]

param(
    [Parameter(Mandatory=$true)]
    [string]$Version,

    [string]$Notes = "",

    [string]$SSHHost = "",
    [string]$SSHUser = "devbox",
    [int]$SSHPort = 2233,
    [string]$RemotePath = "/home/devbox/RPBox/server/releases",
    [string]$ConfigPath = "/home/devbox/RPBox/server/config.yaml"
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$ClientDir = Join-Path $ProjectRoot "client"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  RPBox 客户端发布脚本 v$Version" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# 1. 加载环境变量
Write-Host "`n[1/6] 加载签名密钥..." -ForegroundColor Yellow
$EnvFile = Join-Path $ClientDir ".env"
if (Test-Path $EnvFile) {
    Get-Content $EnvFile | ForEach-Object {
        if ($_ -match "^\s*([^#][^=]+)=(.*)$") {
            $name = $matches[1].Trim()
            $value = $matches[2].Trim()
            [Environment]::SetEnvironmentVariable($name, $value, "Process")
        }
    }
    Write-Host "  已加载 .env 文件" -ForegroundColor Green
} else {
    Write-Host "  错误: 未找到 .env 文件" -ForegroundColor Red
    exit 1
}

# 验证签名密钥
if (-not $env:TAURI_SIGNING_PRIVATE_KEY) {
    Write-Host "  错误: TAURI_SIGNING_PRIVATE_KEY 未设置" -ForegroundColor Red
    exit 1
}
Write-Host "  签名密钥已就绪" -ForegroundColor Green

# 2. 更新版本号
Write-Host "`n[2/6] 更新版本号到 $Version..." -ForegroundColor Yellow

# 更新 tauri.conf.json
$TauriConf = Join-Path $ClientDir "src-tauri\tauri.conf.json"
$TauriJson = Get-Content $TauriConf -Raw | ConvertFrom-Json
$OldVersion = $TauriJson.version
$TauriJson.version = $Version
$TauriJson | ConvertTo-Json -Depth 10 | Set-Content $TauriConf -Encoding UTF8
Write-Host "  tauri.conf.json: $OldVersion -> $Version" -ForegroundColor Green

# 更新 Cargo.toml
$CargoToml = Join-Path $ClientDir "src-tauri\Cargo.toml"
$CargoContent = Get-Content $CargoToml -Raw
$CargoContent = $CargoContent -replace 'version = "[^"]*"', "version = `"$Version`""
Set-Content $CargoToml $CargoContent -Encoding UTF8
Write-Host "  Cargo.toml: 已更新" -ForegroundColor Green

# 更新 package.json
$PackageJson = Join-Path $ClientDir "package.json"
$PkgJson = Get-Content $PackageJson -Raw | ConvertFrom-Json
$PkgJson.version = $Version
$PkgJson | ConvertTo-Json -Depth 10 | Set-Content $PackageJson -Encoding UTF8
Write-Host "  package.json: 已更新" -ForegroundColor Green

# 3. 构建客户端
Write-Host "`n[3/6] 构建 Tauri 客户端..." -ForegroundColor Yellow
Push-Location $ClientDir
try {
    npm run tauri build
    if ($LASTEXITCODE -ne 0) {
        throw "构建失败"
    }
    Write-Host "  构建完成" -ForegroundColor Green
} finally {
    Pop-Location
}

# 4. 收集构建产物
Write-Host "`n[4/6] 收集构建产物..." -ForegroundColor Yellow
$BuildDir = Join-Path $ClientDir "src-tauri\target\release\bundle"
$OutputDir = Join-Path $ProjectRoot "releases\$Version"
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

# Windows NSIS 安装包
$NsisDir = Join-Path $BuildDir "nsis"
$NsisFiles = Get-ChildItem -Path $NsisDir -Filter "*.zip" -ErrorAction SilentlyContinue
foreach ($file in $NsisFiles) {
    Copy-Item $file.FullName $OutputDir
    $SigFile = "$($file.FullName).sig"
    if (Test-Path $SigFile) {
        Copy-Item $SigFile $OutputDir
    }
    Write-Host "  $($file.Name)" -ForegroundColor Green
}

# 生成更新信息文件
$UpdateInfo = @{
    version = $Version
    notes = $Notes
    pub_date = (Get-Date).ToString("yyyy-MM-ddTHH:mm:ssZ")
}
$UpdateInfo | ConvertTo-Json | Set-Content (Join-Path $OutputDir "update.json") -Encoding UTF8
Write-Host "  update.json 已生成" -ForegroundColor Green

# 5. 上传到服务器
Write-Host "`n[5/6] 上传到服务器..." -ForegroundColor Yellow

if (-not $SSHHost) {
    Write-Host "  跳过上传（未配置 SSHHost）" -ForegroundColor Yellow
    Write-Host "  本地产物: $OutputDir" -ForegroundColor Cyan
} else {
    $RemoteVersionDir = "$RemotePath/$Version"

    # 创建远程目录
    ssh -p $SSHPort "${SSHUser}@${SSHHost}" "mkdir -p $RemoteVersionDir"

    # 上传文件
    $FilesToUpload = Get-ChildItem -Path $OutputDir
    foreach ($file in $FilesToUpload) {
        Write-Host "  上传 $($file.Name)..." -ForegroundColor Gray
        scp -P $SSHPort $file.FullName "${SSHUser}@${SSHHost}:${RemoteVersionDir}/"
    }
    Write-Host "  文件上传完成" -ForegroundColor Green
}

# 6. 更新服务器配置
Write-Host "`n[6/6] 更新服务器配置..." -ForegroundColor Yellow

if (-not $SSHHost) {
    Write-Host "  跳过配置更新（未配置 SSHHost）" -ForegroundColor Yellow
} else {
    # 更新 config.yaml 中的版本号
    $UpdateCmd = "sed -i 's/latest_version:.*/latest_version: \`"$Version\`"/' $ConfigPath"
    ssh -p $SSHPort "${SSHUser}@${SSHHost}" $UpdateCmd
    Write-Host "  config.yaml 已更新版本号为 $Version" -ForegroundColor Green
}

# 完成
Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "  发布完成!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "版本: $Version" -ForegroundColor White
Write-Host "本地产物: $OutputDir" -ForegroundColor White
if ($SSHHost) {
    Write-Host "远程路径: ${SSHUser}@${SSHHost}:${RemotePath}/${Version}" -ForegroundColor White
}
Write-Host ""
