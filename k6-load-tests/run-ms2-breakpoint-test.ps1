#!/usr/bin/env pwsh
# MS2 Breaking Point Test Runner
# This script specifically focuses on finding MS2's breaking point

param (
    [string]$ApiKey = "",
    [string]$OutputDir = "./results",
    [switch]$Verbose = $false
)

# Create output directory if it doesn't exist
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
    Write-Host "Created results directory: $OutputDir" -ForegroundColor Green
}

# If no API key is provided, ask for it
if ([string]::IsNullOrEmpty($ApiKey)) {
    $ApiKey = Read-Host "Enter your API key for authentication"
}

if ([string]::IsNullOrEmpty($ApiKey)) {
    Write-Host "ERROR: API key is required to run this test" -ForegroundColor Red
    exit 1
}

# Check if k6 is installed
try {
    $k6Version = k6 version
    Write-Host "Using K6 version: $k6Version" -ForegroundColor Green
} catch {
    Write-Host "K6 is not installed or not in PATH!" -ForegroundColor Red
    Write-Host "Please install K6 from https://k6.io/docs/get-started/installation/" -ForegroundColor Yellow
    exit 1
}

# MS2 endpoint to test
$ms2Endpoint = "http://localhost:8082"

# Check if MS2 is running
Write-Host "Checking if MS2 is running at $ms2Endpoint..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$ms2Endpoint/health" -TimeoutSec 2 -ErrorAction SilentlyContinue
    if ($response.StatusCode -eq 200) {
        Write-Host "✓ MS2 service is running" -ForegroundColor Green
    } else {
        Write-Host "✗ MS2 service returned status $($response.StatusCode)" -ForegroundColor Red
        $confirm = Read-Host "MS2 appears to be having issues. Continue anyway? (y/n)"
        if ($confirm -ne "y") {
            exit 1
        }
    }
} catch {
    Write-Host "✗ MS2 service is not running or not responding" -ForegroundColor Red
    $confirm = Read-Host "MS2 appears to be down. Continue anyway? (y/n)"
    if ($confirm -ne "y") {
        exit 1
    }
}

# Timestamp for results
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$outputFile = "$OutputDir\ms2_breakpoint_${timestamp}.json"

# Build command
$verboseFlag = if ($Verbose) { "--verbose" } else { "" }
$command = "k6 run ./scripts/ms2-queue/ms2-breaking-point-test.js -e API_KEY=$ApiKey $verboseFlag --out json=$outputFile"

# Display test information
Write-Host "======= MS2 BREAKING POINT TEST =======" -ForegroundColor Cyan
Write-Host "Target: MS2 Message Queue Service" -ForegroundColor Green
Write-Host "Description: Determining exact concurrent user capacity" -ForegroundColor Green
Write-Host "API Key: $(if ($ApiKey.Length -gt 8) { $ApiKey.Substring(0, 4) + '...' + $ApiKey.Substring($ApiKey.Length - 4) } else { 'Provided' })" -ForegroundColor Green
Write-Host "Results File: $outputFile" -ForegroundColor Green
Write-Host "=======================================" -ForegroundColor Cyan

Write-Host "`nTest Strategy:" -ForegroundColor Yellow
Write-Host "1. Ramp up from 0 to 3000 VUs in stages" -ForegroundColor White
Write-Host "2. Measure response times and error rates at each stage" -ForegroundColor White
Write-Host "3. Identify the breaking point when error rate exceeds 10%" -ForegroundColor White
Write-Host "4. Save detailed metrics and generate a report" -ForegroundColor White

Write-Host "`nStarting test..." -ForegroundColor Cyan
Write-Host "Command: $command" -ForegroundColor Gray

Invoke-Expression $command

# Check the results file
if (Test-Path "$OutputDir\ms2-breakpoint-summary.json") {
    Write-Host "`nBreaking Point Results:" -ForegroundColor Green
    Get-Content "$OutputDir\ms2-breakpoint-summary.json" | ConvertFrom-Json | Format-List
} else {
    Write-Host "`nNo summary results file found." -ForegroundColor Yellow
}

Write-Host "`nComplete MS2 breaking point test completed. Results saved to $outputFile" -ForegroundColor Green
Write-Host "You can view detailed results in the results directory." -ForegroundColor Green
