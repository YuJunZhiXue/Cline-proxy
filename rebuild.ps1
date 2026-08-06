<#
.SYNOPSIS
    Cline Go Proxy rebuild-and-restart script
.DESCRIPTION
    Stop any running cline-proxy process, rebuild the binary, then start it.
    Designed for development hot-reload workflow.
.PARAMETER Port
    Proxy server port. Default: 3457
.PARAMETER OutputDir
    Output directory. Default: ./dist
.PARAMETER Ldflags
    Linker flags. Default: "-s -w"
.PARAMETER NoStart
    Build only, do not start the process after build
.PARAMETER ProcessName
    Process name pattern to kill before build (without extension). Default: "cline-proxy*"
.EXAMPLE
    .\rebuild.ps1
    Stop running proxy, rebuild, start on port 3457
.EXAMPLE
    .\rebuild.ps1 -Port 3458
    Stop running proxy, rebuild, start on port 3458
.EXAMPLE
    .\rebuild.ps1 -NoStart
    Stop running proxy, rebuild only
.EXAMPLE
    .\rebuild.ps1 -NoOpenBrowser
    Rebuild and start, but do not open browser
.EXAMPLE
    .\rebuild.ps1 -Port 3458 -NoOpenBrowser
    Rebuild on port 3458 without opening browser
#>
[CmdletBinding()]
param(
    [int]$Port = 3457,
    [string]$OutputDir = "",
    [string]$Ldflags = "-s -w",
    [switch]$NoStart,
    [switch]$NoOpenBrowser,
    [string]$ProcessName = "cline-proxy*"
)

$ErrorActionPreference = "Stop"
$projectRoot = $PSScriptRoot
if (-not $projectRoot) { $projectRoot = (Get-Location).Path }
if (-not $OutputDir) { $OutputDir = (Join-Path $projectRoot "dist") }

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " Cline Go Proxy Rebuild" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " Project: $projectRoot"
Write-Host " Port   : $Port"
Write-Host " Output : $OutputDir"
Write-Host ""

# ---------- Step 1: Stop running process ----------
Write-Host "[STEP 1/3] Stopping running process..." -ForegroundColor Yellow
$killed = @()
try {
    $procs = Get-Process -Name $ProcessName -ErrorAction SilentlyContinue
    if ($procs) {
        foreach ($p in $procs) {
            try {
                $killed += "$($p.Name) (pid=$($p.Id))"
                Stop-Process -Id $p.Id -Force -ErrorAction Stop
            } catch {
                Write-Host "  [WARN] Failed to stop pid=$($p.Id): $($_.Exception.Message)" -ForegroundColor DarkYellow
            }
        }
        # Wait for port release
        $waited = 0
        while ($waited -lt 3000) {
            $conn = Get-NetTCPConnection -LocalPort $Port -ErrorAction SilentlyContinue
            if (-not $conn) { break }
            Start-Sleep -Milliseconds 200
            $waited += 200
        }
        Write-Host "  Stopped: $($killed -join ', ')" -ForegroundColor Green
    } else {
        Write-Host "  No running cline-proxy process found." -ForegroundColor DarkGray
    }
} catch {
    # Get-Process or Get-NetTCPConnection may fail on some systems; treat as non-fatal
    Write-Host "  [INFO] Process detection skipped: $($_.Exception.Message)" -ForegroundColor DarkGray
}
Write-Host ""

# ---------- Step 2: Build ----------
Write-Host "[STEP 2/3] Building..." -ForegroundColor Yellow

$goCmd = Get-Command go -ErrorAction SilentlyContinue
if (-not $goCmd) {
    Write-Host "[ERROR] go command not found. Install Go toolchain and add to PATH." -ForegroundColor Red
    exit 1
}

if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

# Build current platform only
$goos = go env GOOS
$goarch = go env GOARCH
$ext = ""
if ($goos -eq "windows") { $ext = ".exe" }
$outName = "cline-proxy-${goos}-${goarch}${ext}"
$outPath = Join-Path $OutputDir $outName

$env:GOOS = $goos
$env:GOARCH = $goarch
$env:CGO_ENABLED = "0"

Write-Host "  Target: $goos/$goarch -> $outName"
& go build -ldflags="$Ldflags" -o $outPath . 2>&1 | ForEach-Object { Write-Host "  $_" }

Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue

if ($LASTEXITCODE -ne 0) {
    Write-Host "[FAIL] Build failed" -ForegroundColor Red
    exit 1
}

if (-not (Test-Path $outPath)) {
    Write-Host "[FAIL] Output not generated: $outPath" -ForegroundColor Red
    exit 1
}

$size = (Get-Item $outPath).Length
$sizeMB = "{0:N2}" -f ($size / 1MB)
Write-Host "[OK] Build succeeded: $outName  size=${sizeMB}MB" -ForegroundColor Green
Write-Host ""

# ---------- Step 3: Start ----------
if ($NoStart) {
    Write-Host "[STEP 3/3] Skipped (-NoStart)" -ForegroundColor DarkGray
    Write-Host ""
    Write-Host "[DONE] Rebuild complete (not started)" -ForegroundColor Green
    exit 0
}

Write-Host "[STEP 3/3] Starting..." -ForegroundColor Yellow
$argList = @("-port", "$Port")
try {
    $started = Start-Process -FilePath $outPath -ArgumentList $argList -WorkingDirectory $projectRoot -PassThru -WindowStyle Hidden
    Start-Sleep -Seconds 1
    if ($started.HasExited) {
        Write-Host "[WARN] Process exited immediately with code $($started.ExitCode)" -ForegroundColor Red
    } else {
        Write-Host "  Started pid=$($started.Id)" -ForegroundColor Green
        # Open browser
        if (-not $NoOpenBrowser) {
            $adminUrl = "http://127.0.0.1:$Port/admin/"
            Start-Sleep -Milliseconds 500
            try {
                Start-Process $adminUrl
                Write-Host "  Browser opened: $adminUrl" -ForegroundColor Green
            } catch {
                Write-Host "  [INFO] Failed to open browser: $($_.Exception.Message)" -ForegroundColor DarkGray
            }
        }
    }
} catch {
    Write-Host "[ERROR] Failed to start: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " Rebuild Complete" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " Proxy : http://127.0.0.1:$Port"
Write-Host " Admin : http://127.0.0.1:$Port/admin/"
Write-Host " Binary: $outPath"
Write-Host "==========================================" -ForegroundColor Cyan
