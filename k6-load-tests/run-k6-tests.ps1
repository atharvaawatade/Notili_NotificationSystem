# NOTLI K6 Load Testing Runner Script
param (
    [string]$Service = "all",
    [string]$Profile = "load",
    [switch]$OutputJson = $false,
    [string]$OutputDir = "./results"
)

# Validate parameters
$validServices = @("all", "ms1", "ms2", "ms3", "ms4", "auth", "queue", "priority", "delivery")
$validProfiles = @("smoke", "load", "stress", "spike", "soak")

if (-not ($validServices -contains $Service)) {
    Write-Host "Invalid service: $Service" -ForegroundColor Red
    Write-Host "Valid options: $($validServices -join ', ')" -ForegroundColor Yellow
    exit 1
}

if (-not ($validProfiles -contains $Profile)) {
    Write-Host "Invalid profile: $Profile" -ForegroundColor Red
    Write-Host "Valid options: $($validProfiles -join ', ')" -ForegroundColor Yellow
    exit 1
}

# Create output directory if it doesn't exist
if ($OutputJson) {
    if (-not (Test-Path $OutputDir)) {
        New-Item -ItemType Directory -Path $OutputDir | Out-Null
    }
}

# Define script path
$scriptPath = ".\scripts\run-all-tests.js"
if ($Service -ne "all") {
    switch ($Service) {
        "ms1" { $scriptPath = ".\scripts\ms1-auth\auth-load-test.js"; break }
        "auth" { $scriptPath = ".\scripts\ms1-auth\auth-load-test.js"; break }
        "ms2" { $scriptPath = ".\scripts\ms2-queue\queue-load-test.js"; break }
        "queue" { $scriptPath = ".\scripts\ms2-queue\queue-load-test.js"; break }
        "ms3" { $scriptPath = ".\scripts\ms3-priority\priority-load-test.js"; break }
        "priority" { $scriptPath = ".\scripts\ms3-priority\priority-load-test.js"; break }
        "ms4" { $scriptPath = ".\scripts\ms4-delivery\delivery-load-test.js"; break }
        "delivery" { $scriptPath = ".\scripts\ms4-delivery\delivery-load-test.js"; break }
    }
}

# Build command
$command = "k6 run $scriptPath -e SERVICE=$Service -e PROFILE=$Profile"

# Add JSON output if requested
if ($OutputJson) {
    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $outputFile = "$OutputDir\${Service}_${Profile}_${timestamp}.json"
    $command += " --out json=$outputFile"
}

# Print test information
Write-Host "======= NOTLI K6 LOAD TESTING =======" -ForegroundColor Cyan
Write-Host "Service: $Service" -ForegroundColor Green
Write-Host "Profile: $Profile" -ForegroundColor Green
if ($OutputJson) {
    Write-Host "Output: $outputFile" -ForegroundColor Green
}
Write-Host "=====================================" -ForegroundColor Cyan

# Check if K6 is installed
try {
    $k6Version = k6 version
    Write-Host "Using K6 version: $k6Version" -ForegroundColor Green
} catch {
    Write-Host "K6 is not installed or not in PATH!" -ForegroundColor Red
    Write-Host "Please install K6 from https://k6.io/docs/get-started/installation/" -ForegroundColor Yellow
    exit 1
}

# Check if all microservices are running (basic check)
$serviceUrls = @{
    "auth" = "http://localhost:8081/health";
    "queue" = "http://localhost:8082/health";
    "priority" = "http://localhost:8083/health";
    "delivery" = "http://localhost:8084/health"
}

$allServicesRunning = $true
if ($Service -eq "all") {
    Write-Host "Checking if microservices are running..." -ForegroundColor Yellow
    foreach ($svc in $serviceUrls.Keys) {
        try {
            $response = Invoke-WebRequest -Uri $serviceUrls[$svc] -TimeoutSec 2 -ErrorAction SilentlyContinue
            if ($response.StatusCode -eq 200) {
                Write-Host "✓ $svc service is running" -ForegroundColor Green
            } else {
                Write-Host "✗ $svc service returned status $($response.StatusCode)" -ForegroundColor Red
                $allServicesRunning = $false
            }
        } catch {
            Write-Host "✗ $svc service is not running" -ForegroundColor Red
            $allServicesRunning = $false
        }
    }

    if (-not $allServicesRunning) {
        Write-Host "Warning: Not all services are running. Tests may fail." -ForegroundColor Yellow
        $confirm = Read-Host "Do you want to continue anyway? (y/n)"
        if ($confirm -ne "y") {
            exit 1
        }
    }
} else {
    # Check only the requested service
    $svcKey = switch ($Service) {
        { $_ -in @("ms1", "auth") } { "auth" }
        { $_ -in @("ms2", "queue") } { "queue" }
        { $_ -in @("ms3", "priority") } { "priority" }
        { $_ -in @("ms4", "delivery") } { "delivery" }
        default { "auth" }
    }
    
    try {
        $response = Invoke-WebRequest -Uri $serviceUrls[$svcKey] -TimeoutSec 2 -ErrorAction SilentlyContinue
        if ($response.StatusCode -eq 200) {
            Write-Host "✓ $svcKey service is running" -ForegroundColor Green
        } else {
            Write-Host "✗ $svcKey service returned status $($response.StatusCode)" -ForegroundColor Red
            $allServicesRunning = $false
        }
    } catch {
        Write-Host "✗ $svcKey service is not running" -ForegroundColor Red
        $allServicesRunning = $false
    }
    
    if (-not $allServicesRunning) {
        Write-Host "Warning: Service is not running. Tests may fail." -ForegroundColor Yellow
        $confirm = Read-Host "Do you want to continue anyway? (y/n)"
        if ($confirm -ne "y") {
            exit 1
        }
    }
}

# Show profile details
Write-Host "Profile Details ($Profile):" -ForegroundColor Cyan
switch ($Profile) {
    "smoke" { 
        Write-Host "  - Virtual Users: 1" -ForegroundColor Yellow
        Write-Host "  - Duration: 10 seconds" -ForegroundColor Yellow
        Write-Host "  - Purpose: Quick verification with minimal load" -ForegroundColor Yellow
    }
    "load" { 
        Write-Host "  - Virtual Users: up to 100" -ForegroundColor Yellow
        Write-Host "  - Duration: 2 minutes" -ForegroundColor Yellow 
        Write-Host "  - Purpose: Normal load simulation" -ForegroundColor Yellow
    }
    "stress" { 
        Write-Host "  - Virtual Users: up to 1200" -ForegroundColor Yellow
        Write-Host "  - Duration: ~13 minutes" -ForegroundColor Yellow
        Write-Host "  - Purpose: High load to find breaking points" -ForegroundColor Yellow
    }
    "spike" { 
        Write-Host "  - Virtual Users: spike to 1500" -ForegroundColor Yellow
        Write-Host "  - Duration: ~3 minutes" -ForegroundColor Yellow
        Write-Host "  - Purpose: Simulate sudden traffic surges" -ForegroundColor Yellow
    }
    "soak" { 
        Write-Host "  - Virtual Users: 400" -ForegroundColor Yellow
        Write-Host "  - Duration: 3 hours" -ForegroundColor Yellow
        Write-Host "  - Purpose: Extended load test for stability" -ForegroundColor Yellow
    }
}

# Run the command
Write-Host "`nStarting test..." -ForegroundColor Cyan
Write-Host "Command: $command" -ForegroundColor Gray
Invoke-Expression $command

# Create a results directory to store test outputs
if (-not (Test-Path "$OutputDir")) {
    New-Item -ItemType Directory -Path "$OutputDir" | Out-Null
}
