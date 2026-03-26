param(
  [string]$Repo = "https://github.com/Konsheng/Sensitive-lexicon.git",
  [string]$Branch = "main"
)

$ErrorActionPreference = "Stop"

$tmpDir = Join-Path $PSScriptRoot "..\\.tmp_sensitive_lexicon_update"
$dst = Join-Path $PSScriptRoot "..\\server\\storage\\moderation\\sensitive_keywords_cn.txt"

if (Test-Path $tmpDir) {
  Remove-Item -Recurse -Force $tmpDir
}

git clone --depth 1 --branch $Branch $Repo $tmpDir | Out-Null

$vocabDir = Join-Path $tmpDir "Vocabulary"
if (-not (Test-Path $vocabDir)) {
  throw "Vocabulary directory not found in upstream repository."
}

$lines = New-Object System.Collections.Generic.List[string]
Get-ChildItem -Path $vocabDir -Filter "*.txt" -File | ForEach-Object {
  Get-Content -Path $_.FullName -Encoding UTF8 | ForEach-Object {
    $v = $_.Trim()
    if ($v -and -not $v.StartsWith("#")) {
      $lines.Add($v)
    }
  }
}

$uniq = $lines | Sort-Object -Unique
$header = @(
  "# Source: https://github.com/Konsheng/Sensitive-lexicon (MIT)"
  "## Auto-generated from Vocabulary/*.txt"
  "## Updated: $(Get-Date -Format 'yyyy-MM-dd')"
  ""
  ""
)

Set-Content -Path $dst -Value $header -Encoding UTF8
Add-Content -Path $dst -Value $uniq -Encoding UTF8

Remove-Item -Recurse -Force $tmpDir

Write-Host "Sensitive keywords updated. Total entries: $($uniq.Count)"
