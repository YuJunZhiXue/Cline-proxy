<#
.SYNOPSIS
    Cline Go Proxy build script
.DESCRIPTION
    Build cline-proxy binary executables. Defaults to current platform.
    Use -Targets for cross-platform compilation; -Clean to wipe output dir.
.PARAMETER Targets
    Target platform list, format "GOOS/GOARCH". Use "current" for host platform.
    Example: "windows/amd64","linux/amd64","darwin/amd64","darwin/arm64"
.PARAMETER OutputDir
    Output directory. Default: ./dist
.PARAMETER Ldflags
    Linker flags. Default: "-s -w" (strip debug info to reduce size)
.PARAMETER Clean
    Clean output directory before build
.EXAMPLE
    .\build.ps1
    Build for current platform only
.EXAMPLE
    .\build.ps1 -Targets "windows/amd64","linux/amd64","darwin/arm64"
    Cross-compile multiple platforms
.EXAMPLE
    .\build.ps1 -Clean
    Clean then build for current platform
#>
[CmdletBinding()]
param(
    [string[]]$Targets = @("current"),
    [string]$OutputDir = "",
    [string]$Ldflags = "-s -w",
    [switch]$Clean
)

$ErrorActionPreference = "Stop"
$projectRoot = $PSScriptRoot
if (-not $projectRoot) { $projectRoot = (Get-Location).Path }
if (-not $OutputDir) { $OutputDir = (Join-Path $projectRoot "dist") }

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " Cline Go Proxy Build" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " Project: $projectRoot"
Write-Host " Output : $OutputDir"
Write-Host ""

# Check go command
$goCmd = Get-Command go -ErrorAction SilentlyContinue
if (-not $goCmd) {
    Write-Host "[ERROR] go command not found. Install Go toolchain and add to PATH." -ForegroundColor Red
    exit 1
}
Write-Host " Go: $($goCmd.Source)"
$goVersion = (go version) 2>&1
Write-Host " Version: $goVersion"
Write-Host ""

# Clean output dir
if ($Clean -and (Test-Path $OutputDir)) {
    Write-Host "[INFO] Cleaning output dir: $OutputDir" -ForegroundColor Yellow
    Remove-Item -Recurse -Force $OutputDir
}

if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

# Resolve targets
$resolvedTargets = @()
foreach ($t in $Targets) {
    if ($t -eq "current") {
        $goos = $env:GOOS
        if (-not $goos) { $goos = go env GOOS }
        $goarch = $env:GOARCH
        if (-not $goarch) { $goarch = go env GOARCH }
        $resolvedTargets += "$goos/$goarch"
    }
    else {
        $resolvedTargets += $t
    }
}

Write-Host " Targets: $($resolvedTargets -join ', ')"
Write-Host " Ldflags : $Ldflags"
Write-Host "------------------------------------------"

$failed = @()
$succeeded = @()

foreach ($target in $resolvedTargets) {
    $parts = $target -split "/"
    if ($parts.Count -ne 2) {
        Write-Host "[WARN] Skipping invalid target: $target (expected GOOS/GOARCH)" -ForegroundColor Yellow
        continue
    }
    $goos = $parts[0]
    $goarch = $parts[1]

    $ext = ""
    if ($goos -eq "windows") { $ext = ".exe" }
    $outName = "cline-proxy-${goos}-${goarch}${ext}"
    $outPath = Join-Path $OutputDir $outName

    Write-Host ""
    Write-Host "[BUILD] $goos/$goarch -> $outName" -ForegroundColor Green

    $env:GOOS = $goos
    $env:GOARCH = $goarch
    $env:CGO_ENABLED = "0"

    & go build -ldflags="$Ldflags" -o $outPath . 2>&1 | ForEach-Object { Write-Host "  $_" }

    if ($LASTEXITCODE -ne 0) {
        Write-Host "[FAIL] $target build failed" -ForegroundColor Red
        $failed += $target
        continue
    }

    if (Test-Path $outPath) {
        $size = (Get-Item $outPath).Length
        $sizeMB = "{0:N2}" -f ($size / 1MB)
        Write-Host "[OK]   $target -> $outName  size=${sizeMB}MB" -ForegroundColor Green
        $succeeded += [PSCustomObject]@{ Target = $target; Path = $outPath; SizeMB = $sizeMB }
    }
    else {
        Write-Host "[FAIL] $target output not generated" -ForegroundColor Red
        $failed += $target
    }
}

# Restore env
Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " Build Summary" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
if ($succeeded.Count -gt 0) {
    Write-Host " Succeeded ($($succeeded.Count)):" -ForegroundColor Green
    $succeeded | ForEach-Object { Write-Host ("   - {0,-20} {1,8} MB  {2}" -f $_.Target, $_.SizeMB, $_.Path) }
}
if ($failed.Count -gt 0) {
    Write-Host " Failed ($($failed.Count)):" -ForegroundColor Red
    $failed | ForEach-Object { Write-Host "   - $_" }
}
Write-Host ""

if ($failed.Count -gt 0) { exit 1 }
Write-Host "[DONE] Build complete" -ForegroundColor Green
