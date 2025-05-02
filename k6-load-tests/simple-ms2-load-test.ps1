#!/usr/bin/env pwsh
# Simple MS2 Breaking Point Load Test 
# This PowerShell script tests the MS2 breaking point without requiring K6

param (
    [string]$ApiKey = "",
    [int]$StartConcurrency = 100,
    [int]$MaxConcurrency = 3000,
    [int]$ConcurrencyStep = 100,
    [int]$RequestsPerConcurrencyLevel = 100,
    [string]$OutputFile = "./results/ms2-breakpoint-$(Get-Date -Format 'yyyyMMdd-HHmmss').json",
    [string]$TargetUrl = "http://localhost:8082/v1/email/notify"
)

# Create results directory if it doesn't exist
$resultsDir = Split-Path -Parent $OutputFile
if (-not (Test-Path $resultsDir)) {
    New-Item -ItemType Directory -Path $resultsDir | Out-Null
    Write-Host "Created results directory: $resultsDir" -ForegroundColor Green
}

# Request API key if not provided
if ([string]::IsNullOrEmpty($ApiKey)) {
    $ApiKey = Read-Host "Enter your API key for authentication"
}

if ([string]::IsNullOrEmpty($ApiKey)) {
    Write-Host "ERROR: API key is required to run this test" -ForegroundColor Red
    exit 1
}

# Test if MS2 is running
Write-Host "Checking if MS2 is running..." -ForegroundColor Yellow
try {
    $healthResponse = Invoke-WebRequest -Uri "http://localhost:8082/health" -TimeoutSec 2 -ErrorAction SilentlyContinue
    if ($healthResponse.StatusCode -eq 200) {
        Write-Host "✓ MS2 service is running" -ForegroundColor Green
    } else {
        Write-Host "✗ MS2 service returned status $($healthResponse.StatusCode)" -ForegroundColor Red
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

# Function to generate a random UUID
function New-UUID {
    return [guid]::NewGuid().ToString()
}

# Function to generate random email
function New-RandomEmail {
    $username = -join ((65..90) + (97..122) | Get-Random -Count 8 | ForEach-Object { [char]$_ })
    $domain = -join ((97..122) | Get-Random -Count 6 | ForEach-Object { [char]$_ })
    return "$username@$domain.com"
}

# Function to send a single request to MS2
function Send-MS2Request {
    param (
        [string]$ApiKey,
        [string]$TargetUrl
    )
    
    $idempotencyKey = New-UUID
    $payload = @{
        idempotency_key = $idempotencyKey
        recipient = New-RandomEmail
        subject = "Load Test $idempotencyKey"
        body = "This is a load test email for MS2 breaking point detection."
        from_name = "MS2 Load Test"
        reply_to = @("noreply@notli.test")
        channel_data = @{
            attachments = @()
            cc = @()
            bcc = @()
        }
    } | ConvertTo-Json
    
    $headers = @{
        "Content-Type" = "application/json"
        "X-API-Key" = $ApiKey
    }
    
    $startTime = Get-Date
    
    try {
        $response = Invoke-WebRequest -Uri $TargetUrl -Method Post -Body $payload -Headers $headers -TimeoutSec 10 -ErrorAction SilentlyContinue
        $statusCode = $response.StatusCode
        $body = $response.Content
    } catch {
        $statusCode = $_.Exception.Response.StatusCode.value__
        $body = $_.Exception.Message
    }
    
    $endTime = Get-Date
    $duration = ($endTime - $startTime).TotalMilliseconds
    
    return @{
        StatusCode = $statusCode
        Duration = $duration
        IdempotencyKey = $idempotencyKey
        Success = ($statusCode -ge 200 -and $statusCode -lt 300)
    }
}

# Function to run concurrent requests
function Invoke-ConcurrentRequests {
    param (
        [int]$Concurrency,
        [int]$TotalRequests,
        [string]$ApiKey,
        [string]$TargetUrl
    )
    
    Write-Host "Running $TotalRequests requests at concurrency level $Concurrency..." -ForegroundColor Yellow
    
    $runspacePool = [runspacefactory]::CreateRunspacePool(1, $Concurrency)
    $runspacePool.Open()
    
    $scriptBlock = {
        param($apiKey, $targetUrl)
        
        # Define these functions inside the script block for the runspace
        function New-UUID {
            return [guid]::NewGuid().ToString()
        }

        function New-RandomEmail {
            $username = -join ((65..90) + (97..122) | Get-Random -Count 8 | ForEach-Object { [char]$_ })
            $domain = -join ((97..122) | Get-Random -Count 6 | ForEach-Object { [char]$_ })
            return "$username@$domain.com"
        }
        
        $idempotencyKey = New-UUID
        $payload = @{
            idempotency_key = $idempotencyKey
            recipient = New-RandomEmail
            subject = "Load Test $idempotencyKey"
            body = "This is a load test email for MS2 breaking point detection."
            from_name = "MS2 Load Test"
            reply_to = @("noreply@notli.test")
            channel_data = @{
                attachments = @()
                cc = @()
                bcc = @()
            }
        } | ConvertTo-Json
        
        $headers = @{
            "Content-Type" = "application/json"
            "X-API-Key" = $apiKey
        }
        
        $startTime = Get-Date
        
        try {
            $response = Invoke-WebRequest -Uri $targetUrl -Method Post -Body $payload -Headers $headers -TimeoutSec 10 -ErrorAction SilentlyContinue
            $statusCode = $response.StatusCode
            $body = $response.Content
        } catch {
            $statusCode = $_.Exception.Response.StatusCode.value__
            $body = $_.Exception.Message
        }
        
        $endTime = Get-Date
        $duration = ($endTime - $startTime).TotalMilliseconds
        
        return @{
            StatusCode = $statusCode
            Duration = $duration
            IdempotencyKey = $idempotencyKey
            Success = ($statusCode -ge 200 -and $statusCode -lt 300)
        }
    }
    
    $jobs = @()
    for ($i = 0; $i -lt $TotalRequests; $i++) {
        $ps = [powershell]::Create().AddScript($scriptBlock).AddArgument($ApiKey).AddArgument($TargetUrl)
        $ps.RunspacePool = $runspacePool
        
        $jobs += @{
            Powershell = $ps
            Handle = $ps.BeginInvoke()
        }
    }
    
    $results = @()
    foreach ($job in $jobs) {
        $results += $job.Powershell.EndInvoke($job.Handle)
        $job.Powershell.Dispose()
    }
    
    $runspacePool.Close()
    $runspacePool.Dispose()
    
    return $results
}

# Main test logic
$results = @{}
$breakingPointFound = $false
$breakingPointConcurrency = 0
$breakingPointErrorRate = 0

Write-Host "=============================================" -ForegroundColor Cyan
Write-Host "   MS2 BREAKING POINT TEST" -ForegroundColor Cyan
Write-Host "=============================================" -ForegroundColor Cyan
Write-Host "API Key: $(if ($ApiKey.Length -gt 8) { $ApiKey.Substring(0, 4) + '...' + $ApiKey.Substring($ApiKey.Length - 4) } else { 'Provided' })" -ForegroundColor Green
Write-Host "Target URL: $TargetUrl" -ForegroundColor Green
Write-Host "Starting Concurrency: $StartConcurrency" -ForegroundColor Green
Write-Host "Maximum Concurrency: $MaxConcurrency" -ForegroundColor Green
Write-Host "Concurrency Step: $ConcurrencyStep" -ForegroundColor Green
Write-Host "Requests per level: $RequestsPerConcurrencyLevel" -ForegroundColor Green
Write-Host "Output File: $OutputFile" -ForegroundColor Green
Write-Host "=============================================" -ForegroundColor Cyan

for ($concurrency = $StartConcurrency; $concurrency -le $MaxConcurrency -and -not $breakingPointFound; $concurrency += $ConcurrencyStep) {
    Write-Host "Testing concurrency level: $concurrency" -ForegroundColor Yellow
    
    # Run concurrent requests
    $concurrencyResults = Invoke-ConcurrentRequests -Concurrency $concurrency -TotalRequests $RequestsPerConcurrencyLevel -ApiKey $ApiKey -TargetUrl $TargetUrl
    
    # Calculate metrics
    $successCount = ($concurrencyResults | Where-Object { $_.Success } | Measure-Object).Count
    $errorCount = $RequestsPerConcurrencyLevel - $successCount
    $errorRate = $errorCount / $RequestsPerConcurrencyLevel
    
    $p95Duration = $concurrencyResults.Duration | Sort-Object | Select-Object -Index ([math]::Floor($RequestsPerConcurrencyLevel * 0.95))
    $medianDuration = $concurrencyResults.Duration | Sort-Object | Select-Object -Index ([math]::Floor($RequestsPerConcurrencyLevel * 0.5))
    $maxDuration = ($concurrencyResults.Duration | Measure-Object -Maximum).Maximum
    $avgDuration = ($concurrencyResults.Duration | Measure-Object -Average).Average
    
    # Store results
    $results[$concurrency] = @{
        Concurrency = $concurrency
        TotalRequests = $RequestsPerConcurrencyLevel
        SuccessCount = $successCount
        ErrorCount = $errorCount
        ErrorRate = $errorRate
        MedianResponseTime = $medianDuration
        P95ResponseTime = $p95Duration
        MaxResponseTime = $maxDuration
        AvgResponseTime = $avgDuration
    }
    
    # Display current results
    Write-Host "  Completed $RequestsPerConcurrencyLevel requests with $concurrency concurrent users" -ForegroundColor White
    Write-Host "  Success Rate: $(100 - ($errorRate * 100))%" -ForegroundColor $(if ($errorRate -lt 0.1) { "Green" } elseif ($errorRate -lt 0.2) { "Yellow" } else { "Red" })
    Write-Host "  Median Response: $([math]::Round($medianDuration, 2))ms" -ForegroundColor White
    Write-Host "  P95 Response: $([math]::Round($p95Duration, 2))ms" -ForegroundColor White
    Write-Host "  Max Response: $([math]::Round($maxDuration, 2))ms" -ForegroundColor White
    
    # Check for breaking point (error rate > 10%)
    if ($errorRate -gt 0.1 -and $concurrency -gt $StartConcurrency) {
        $breakingPointFound = $true
        $breakingPointConcurrency = $concurrency
        $breakingPointErrorRate = $errorRate
        
        Write-Host "`n================ BREAKING POINT DETECTED =================" -ForegroundColor Red
        Write-Host "Breaking Point: $breakingPointConcurrency concurrent users" -ForegroundColor Red
        Write-Host "Error Rate: $([math]::Round($breakingPointErrorRate * 100, 2))%" -ForegroundColor Red
        Write-Host "===========================================================`n" -ForegroundColor Red
        
        # One more test to confirm
        Write-Host "Running confirmation test at breaking point..." -ForegroundColor Yellow
        $confirmResults = Invoke-ConcurrentRequests -Concurrency $breakingPointConcurrency -TotalRequests $RequestsPerConcurrencyLevel -ApiKey $ApiKey -TargetUrl $TargetUrl
        $confirmSuccessCount = ($confirmResults | Where-Object { $_.Success } | Measure-Object).Count
        $confirmErrorRate = ($RequestsPerConcurrencyLevel - $confirmSuccessCount) / $RequestsPerConcurrencyLevel
        
        Write-Host "Confirmation test error rate: $([math]::Round($confirmErrorRate * 100, 2))%" -ForegroundColor Yellow
    }
    
    # Wait a bit between tests to allow system to recover
    if (-not $breakingPointFound) {
        Write-Host "Waiting 5 seconds before next concurrency level..." -ForegroundColor Gray
        Start-Sleep -Seconds 5
    }
}

# If we hit max concurrency without breaking point
if (-not $breakingPointFound) {
    Write-Host "`n======== NO BREAKING POINT DETECTED ========" -ForegroundColor Green
    Write-Host "MS2 successfully handled up to $MaxConcurrency concurrent users!" -ForegroundColor Green
    Write-Host "You may want to run the test again with a higher maximum concurrency." -ForegroundColor Green
    Write-Host "===========================================`n" -ForegroundColor Green
}

# Prepare summary
$summary = @{
    TestDate = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    BreakingPointFound = $breakingPointFound
    BreakingPointConcurrency = if ($breakingPointFound) { $breakingPointConcurrency } else { "Not found up to $MaxConcurrency" }
    BreakingPointErrorRate = if ($breakingPointFound) { $breakingPointErrorRate * 100 } else { 0 }
    ConcurrencyLevels = $results
}

# Save results to file
$summary | ConvertTo-Json -Depth 4 | Out-File -FilePath $OutputFile
Write-Host "Results saved to: $OutputFile" -ForegroundColor Green

# Display final summary
Write-Host "`n============ MS2 BREAKING POINT TEST RESULTS ============" -ForegroundColor Cyan
if ($breakingPointFound) {
    Write-Host "MS2 Breaking Point: $breakingPointConcurrency concurrent users" -ForegroundColor $(if ($breakingPointConcurrency -ge 1000) { "Green" } else { "Yellow" })
    Write-Host "Error Rate at Breaking Point: $([math]::Round($breakingPointErrorRate * 100, 2))%" -ForegroundColor Yellow
} else {
    Write-Host "MS2 successfully handled all tests up to $MaxConcurrency concurrent users!" -ForegroundColor Green
}

Write-Host "`nPerformance by Concurrency Level:" -ForegroundColor White
foreach ($level in ($results.Keys | Sort-Object)) {
    $r = $results[$level]
    Write-Host "  $($r.Concurrency) users: $([math]::Round(100 - ($r.ErrorRate * 100), 1))% success, $([math]::Round($r.MedianResponseTime, 0))ms median, $([math]::Round($r.P95ResponseTime, 0))ms p95" -ForegroundColor $(if ($r.ErrorRate -lt 0.05) { "Green" } elseif ($r.ErrorRate -lt 0.1) { "Yellow" } else { "Red" })
}

Write-Host "`nComplete results saved to: $OutputFile" -ForegroundColor Cyan
Write-Host "===============================================" -ForegroundColor Cyan
