# Simple MS2 Load Test Script
param (
    [string]$ApiKey,
    [int]$MaxConcurrent = 2000,
    [int]$StepSize = 200
)

# Create results directory if it doesn't exist
$resultsDir = "./results"
if (-not (Test-Path $resultsDir)) {
    New-Item -ItemType Directory -Path $resultsDir | Out-Null
}

# Validate API key
if ([string]::IsNullOrEmpty($ApiKey)) {
    Write-Host "Error: API key is required" -ForegroundColor Red
    exit 1
}

# Test MS2 health
try {
    $health = Invoke-WebRequest -Uri "http://localhost:8082/health" -ErrorAction SilentlyContinue
    Write-Host "MS2 is running: $($health.StatusCode)" -ForegroundColor Green
}
catch {
    Write-Host "Warning: MS2 may not be running. Test may fail." -ForegroundColor Yellow
}

# Function to test MS2 with concurrent requests
function Test-MS2Concurrency {
    param (
        [int]$Concurrency,
        [string]$ApiKey
    )
    
    Write-Host "Testing with $Concurrency concurrent requests..." -ForegroundColor Yellow
    
    $successful = 0
    $failed = 0
    $times = @()
    
    # Create runspace pool for parallel execution
    $runspacePool = [runspacefactory]::CreateRunspacePool(1, $Concurrency)
    $runspacePool.Open()
    
    $jobs = @()
    
    # Create and start jobs
    for ($i = 0; $i -lt $Concurrency; $i++) {
        $ps = [powershell]::Create().AddScript({
            param($apiKey, $id)
            
            $idempotencyKey = [guid]::NewGuid().ToString()
            $email = "test$id@example.com"
            
            $body = @{
                idempotency_key = $idempotencyKey
                recipient = $email
                subject = "Load Test $id"
                body = "This is a load test email $id"
                from_name = "MS2 Test"
                reply_to = @("test@example.com")
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
            
            $start = Get-Date
            
            try {
                $response = Invoke-WebRequest -Uri "http://localhost:8082/v1/email/notify" -Method Post -Body $body -Headers $headers -TimeoutSec 30
                $success = $true
                $status = $response.StatusCode
            }
            catch {
                $success = $false
                $status = $_.Exception.Response.StatusCode.value__
            }
            
            $end = Get-Date
            $duration = ($end - $start).TotalMilliseconds
            
            return @{
                Success = $success
                Status = $status
                Duration = $duration
            }
        }).AddArgument($ApiKey).AddArgument($i)
        
        $ps.RunspacePool = $runspacePool
        
        $jobs += @{
            PowerShell = $ps
            Handle = $ps.BeginInvoke()
            Id = $i
        }
    }
    
    # Collect results
    foreach ($job in $jobs) {
        $result = $job.PowerShell.EndInvoke($job.Handle)
        $job.PowerShell.Dispose()
        
        if ($result.Success) {
            $successful++
        }
        else {
            $failed++
        }
        
        $times += $result.Duration
    }
    
    $runspacePool.Close()
    $runspacePool.Dispose()
    
    # Calculate statistics
    $successRate = ($successful / $Concurrency) * 100
    $avgTime = ($times | Measure-Object -Average).Average
    $maxTime = ($times | Measure-Object -Maximum).Maximum
    $p95Time = $times | Sort-Object | Select-Object -Index ([Math]::Floor($times.Count * 0.95))
    
    $result = @{
        Concurrency = $Concurrency
        SuccessCount = $successful
        FailCount = $failed
        SuccessRate = $successRate
        AvgResponseTime = $avgTime
        MaxResponseTime = $maxTime
        P95ResponseTime = $p95Time
        BreakingPoint = ($successRate -lt 90)
    }
    
    Write-Host "  Success Rate: $([Math]::Round($successRate, 2))%" -ForegroundColor $(if ($successRate -ge 90) { "Green" } else { "Red" })
    Write-Host "  Avg Response: $([Math]::Round($avgTime, 2))ms" -ForegroundColor White
    Write-Host "  P95 Response: $([Math]::Round($p95Time, 2))ms" -ForegroundColor White
    Write-Host "  Max Response: $([Math]::Round($maxTime, 2))ms" -ForegroundColor White
    
    return $result
}

# Run the load test
$results = @()
$breakingPoint = $null

Write-Host "MS2 Breaking Point Test" -ForegroundColor Cyan
Write-Host "Testing up to $MaxConcurrent concurrent users in steps of $StepSize" -ForegroundColor Cyan
Write-Host "----------------------------------------------" -ForegroundColor Cyan

# Test incrementally with increasing concurrency
for ($concurrency = $StepSize; $concurrency -le $MaxConcurrent; $concurrency += $StepSize) {
    $result = Test-MS2Concurrency -Concurrency $concurrency -ApiKey $ApiKey
    $results += $result
    
    # If this is the breaking point and we haven't found one yet
    if ($result.BreakingPoint -and -not $breakingPoint) {
        $breakingPoint = $concurrency
        Write-Host "BREAKING POINT DETECTED: $breakingPoint concurrent users" -ForegroundColor Red
        
        # Run one more test to confirm
        Write-Host "Running confirmation test..." -ForegroundColor Yellow
        $confirmResult = Test-MS2Concurrency -Concurrency $concurrency -ApiKey $ApiKey
        
        if ($confirmResult.BreakingPoint) {
            Write-Host "Breaking point confirmed at $breakingPoint concurrent users" -ForegroundColor Red
        }
        else {
            Write-Host "Breaking point not confirmed on second test" -ForegroundColor Yellow
            $breakingPoint = $null
        }
    }
    
    # If we've reached a breaking point, no need to test higher concurrency
    if ($breakingPoint) {
        break
    }
    
    # Give the system a moment to recover
    Start-Sleep -Seconds 3
}

# If we never found a breaking point
if (-not $breakingPoint) {
    Write-Host "No breaking point detected up to $MaxConcurrent concurrent users" -ForegroundColor Green
}

# Save results
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$outputFile = "$resultsDir/ms2-results-$timestamp.json"

$finalResults = @{
    TestDate = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    BreakingPoint = $breakingPoint
    MaxConcurrencyTested = if ($breakingPoint) { $breakingPoint } else { $MaxConcurrent }
    ConcurrencyResults = $results
}

$finalResults | ConvertTo-Json -Depth 4 | Out-File -FilePath $outputFile

Write-Host "Test complete! Results saved to $outputFile" -ForegroundColor Green
Write-Host "Summary:" -ForegroundColor Cyan
if ($breakingPoint) {
    Write-Host "MS2 breaking point: $breakingPoint concurrent users" -ForegroundColor Yellow
}
else {
    Write-Host "MS2 handled up to $MaxConcurrent concurrent users without breaking" -ForegroundColor Green
}

# Show performance at different concurrency levels
Write-Host "Performance by concurrency level:" -ForegroundColor Cyan
foreach ($result in $results) {
    Write-Host "$($result.Concurrency) users: $([Math]::Round($result.SuccessRate, 2))% success, $([Math]::Round($result.AvgResponseTime, 0))ms avg, $([Math]::Round($result.P95ResponseTime, 0))ms p95" -ForegroundColor $(if ($result.SuccessRate -ge 90) { "Green" } else { "Red" })
}
