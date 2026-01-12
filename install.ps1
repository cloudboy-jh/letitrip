param(
  [string]$InstallDir = "$env:LOCALAPPDATA\letitrip\bin",
  [string]$Repo = "cloudboy-jh/letitrip",
  [string]$Version = "latest"
)

$ErrorActionPreference = "Stop"
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

Write-Host "Installing letitrip..." -ForegroundColor Cyan

# Create install directory
if (-not (Test-Path -Path $InstallDir)) {
  New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# Get latest release
$releaseUrl = if ($Version -eq "latest") {
  "https://api.github.com/repos/$Repo/releases/latest"
} else {
  "https://api.github.com/repos/$Repo/releases/tags/$Version"
}

Write-Host "Fetching release information..." -ForegroundColor Gray
$release = Invoke-RestMethod -Uri $releaseUrl -Headers @{ "User-Agent" = "letitrip-installer" }
$asset = $release.assets | Where-Object { $_.name -eq "letitrip_windows_amd64.zip" } | Select-Object -First 1

if (-not $asset) {
  throw "Release asset letitrip_windows_amd64.zip not found for $Repo ($Version)."
}

# Download and extract
Write-Host "Downloading letitrip v$($release.tag_name)..." -ForegroundColor Gray
$tempZip = Join-Path $env:TEMP "letitrip_windows_amd64.zip"
Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $tempZip

Write-Host "Extracting..." -ForegroundColor Gray
Expand-Archive -Path $tempZip -DestinationPath $InstallDir -Force
Remove-Item $tempZip -Force

$exePath = Join-Path $InstallDir "letitrip.exe"
if (-not (Test-Path -Path $exePath)) {
  throw "letitrip.exe not found in $InstallDir after extraction."
}

# Add to PATH
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*${InstallDir}*") {
  Write-Host "Adding to PATH..." -ForegroundColor Gray
  [Environment]::SetEnvironmentVariable("Path", "$currentPath;$InstallDir", "User")
  $env:Path = "$env:Path;$InstallDir"
}

# Check for FLAC tools
Write-Host "`nChecking for FLAC tools..." -ForegroundColor Cyan
$flacInstalled = $false
try {
  $null = Get-Command flac.exe -ErrorAction Stop
  $null = Get-Command metaflac.exe -ErrorAction Stop
  $flacInstalled = $true
  Write-Host "✓ FLAC tools already installed" -ForegroundColor Green
} catch {
  Write-Host "✗ FLAC tools not found" -ForegroundColor Yellow
}

if (-not $flacInstalled) {
  Write-Host "`nInstalling FLAC tools..." -ForegroundColor Cyan
  try {
    winget install --id Xiph.FLAC --silent --accept-package-agreements --accept-source-agreements
    Write-Host "✓ FLAC tools installed" -ForegroundColor Green
  } catch {
    Write-Host "⚠ Failed to install FLAC tools automatically" -ForegroundColor Yellow
    Write-Host "Please install manually: winget install Xiph.FLAC" -ForegroundColor Yellow
    Write-Host "Or download from: https://xiph.org/flac/download.html" -ForegroundColor Yellow
  }
}

Write-Host "`n✓ letitrip installed successfully to $InstallDir" -ForegroundColor Green
Write-Host "`nUsage:" -ForegroundColor Cyan
Write-Host "  letitrip                  # Rip the current CD"
Write-Host "  letitrip --info           # Show disc info"
Write-Host "  letitrip --output D:\Music # Specify output directory"
Write-Host "`nNote: You may need to restart your terminal for PATH changes to take effect." -ForegroundColor Yellow
