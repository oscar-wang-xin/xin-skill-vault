param(
  [string]$Version = "0.1.0"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

wails3 build
if ($LASTEXITCODE -ne 0) { throw "wails3 build failed" }

$releaseRoot = Join-Path $root "bin\release"
$stage = Join-Path $releaseRoot "XinSkillVault-$Version-win-x64"
if (Test-Path $stage) { Remove-Item -Recurse -Force $stage }
New-Item -ItemType Directory -Path $stage -Force | Out-Null

Copy-Item (Join-Path $root "bin\xinskillvault.exe") $stage
Copy-Item (Join-Path $root "README.md") $stage
Copy-Item (Join-Path $root "LICENSE") $stage
$zip = Join-Path $releaseRoot "XinSkillVault-$Version-win-x64.zip"
if (Test-Path $zip) { Remove-Item -Force $zip }
Compress-Archive -Path (Join-Path $stage "*") -DestinationPath $zip -CompressionLevel Optimal

$hash = Get-FileHash $zip -Algorithm SHA256
Write-Host "Artifact: $zip"
Write-Host "SHA256:   $($hash.Hash)"
