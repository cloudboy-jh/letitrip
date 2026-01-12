# Emergency fix script - builds from source directly
$ErrorActionPreference = "Stop"

Write-Host "=== LetItRip Emergency Installer ===" -ForegroundColor Cyan
Write-Host "This will build from source to ensure you get the Windows version`n" -ForegroundColor Yellow

# Remove any old cached versions
Write-Host "Removing old versions..." -ForegroundColor Gray
$oldPaths = @(
    "$env:USERPROFILE\go\bin\letitrip.exe",
    "$env:LOCALAPPDATA\letitrip\bin\letitrip.exe"
)
foreach ($path in $oldPaths) {
    if (Test-Path $path) {
        Remove-Item $path -Force
        Write-Host "  Removed: $path" -ForegroundColor Gray
    }
}

# Set up install directory
$InstallDir = "$env:LOCALAPPDATA\letitrip\bin"
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# Clone and build
Write-Host "`nBuilding from source..." -ForegroundColor Cyan
$TempDir = Join-Path $env:TEMP "letitrip-build-$(Get-Random)"
try {
    # Clone repo
    Write-Host "  Cloning repository..." -ForegroundColor Gray
    git clone --depth 1 https://github.com/cloudboy-jh/letitrip.git $TempDir 2>&1 | Out-Null
    
    # Build
    Write-Host "  Building Windows binary..." -ForegroundColor Gray
    Push-Location "$TempDir\letitrip"
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    go build -o letitrip.exe .
    
    # Check that we built the Windows version
    $builtForWindows = $true
    if (-not (Test-Path "letitrip.exe")) {
        throw "Build failed - letitrip.exe not found"
    }
    
    # Copy to install location
    Copy-Item "letitrip.exe" $InstallDir -Force
    Pop-Location
    
    Write-Host "  ✓ Built successfully" -ForegroundColor Green
} finally {
    # Cleanup
    if (Test-Path $TempDir) {
        Remove-Item $TempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# Add to PATH
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$InstallDir*") {
    Write-Host "`nAdding to PATH..." -ForegroundColor Gray
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
}

# Check for FLAC
Write-Host "`nChecking FLAC tools..." -ForegroundColor Cyan
try {
    $null = Get-Command flac.exe -ErrorAction Stop
    $null = Get-Command metaflac.exe -ErrorAction Stop
    Write-Host "  ✓ FLAC tools found" -ForegroundColor Green
} catch {
    Write-Host "  ⚠ FLAC tools not found" -ForegroundColor Yellow
    Write-Host "  Installing FLAC..." -ForegroundColor Gray
    try {
        winget install --id Xiph.FLAC --silent --accept-package-agreements --accept-source-agreements
        Write-Host "  ✓ FLAC installed" -ForegroundColor Green
    } catch {
        Write-Host "  Please install manually: winget install Xiph.FLAC" -ForegroundColor Yellow
    }
}

Write-Host "`n=== Installation Complete ===" -ForegroundColor Green
Write-Host "Location: $InstallDir\letitrip.exe" -ForegroundColor Gray
Write-Host "`nTest it:" -ForegroundColor Cyan
Write-Host "  letitrip --info" -ForegroundColor White
Write-Host "`nNote: If 'letitrip' is not recognized, restart your terminal" -ForegroundColor Yellow
