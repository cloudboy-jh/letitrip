param(
  [string]$InstallDir = "$env:LOCALAPPDATA\letitrip\bin",
  [string]$Repo = "cloudboy-jh/letitrip",
  [string]$Version = "latest"
)

$ErrorActionPreference = "Stop"
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

if (-not (Test-Path -Path $InstallDir)) {
  New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$releaseUrl = if ($Version -eq "latest") {
  "https://api.github.com/repos/$Repo/releases/latest"
} else {
  "https://api.github.com/repos/$Repo/releases/tags/$Version"
}

$release = Invoke-RestMethod -Uri $releaseUrl -Headers @{ "User-Agent" = "letitrip-installer" }
$asset = $release.assets | Where-Object { $_.name -eq "letitrip_windows_amd64.zip" } | Select-Object -First 1

if (-not $asset) {
  throw "Release asset letitrip_windows_amd64.zip not found for $Repo ($Version)."
}

$tempZip = Join-Path $env:TEMP "letitrip_windows_amd64.zip"
Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $tempZip

Expand-Archive -Path $tempZip -DestinationPath $InstallDir -Force
Remove-Item $tempZip -Force

$exePath = Join-Path $InstallDir "letitrip.exe"
if (-not (Test-Path -Path $exePath)) {
  throw "letitrip.exe not found in $InstallDir after extraction."
}

$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*${InstallDir}*") {
  [Environment]::SetEnvironmentVariable("Path", "$currentPath;$InstallDir", "User")
}

Write-Host "letitrip installed to $InstallDir"
Write-Host "Open a new terminal and run: letitrip"