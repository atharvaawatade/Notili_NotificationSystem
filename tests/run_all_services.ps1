# PowerShell script to start all microservices for integration testing

# Create a function to start each service in a new terminal
function Start-Service {
    param (
        [string]$ServiceName,
        [string]$Directory,
        [string]$Command
    )
    
    Write-Host "Starting $ServiceName..." -ForegroundColor Green
    Start-Process powershell -ArgumentList "-NoExit -Command `"cd '$Directory'; $Command`""
}

# Set environment variables for logging
$env:NOTLI_LOG_SERVER = "http://localhost:3005"

# Start Auth Service (MS1) - Port 3000
Start-Service -ServiceName "Auth Service" -Directory "D:\Appointy_Projects\Notli\microservice1_authentication" -Command "go run cmd/server/main.go"

# Wait for Auth Service to initialize
Write-Host "Waiting for Auth Service to initialize..."
Start-Sleep -Seconds 5

# Start Message Queue Service (MS2) - Port 3001
Start-Service -ServiceName "Message Queue Service" -Directory "D:\Appointy_Projects\Notli\microservice2_messageq" -Command "go run cmd/server/main.go"

# Wait for Message Queue Service to initialize
Write-Host "Waiting for Message Queue Service to initialize..."
Start-Sleep -Seconds 5

# Start Prioritization Service (MS3) - Port 3002
Start-Service -ServiceName "Prioritization Service" -Directory "D:\Appointy_Projects\Notli\microservice3_prioritization" -Command "go run cmd/server/main.go"

# Wait for Prioritization Service to initialize
Write-Host "Waiting for Prioritization Service to initialize..."
Start-Sleep -Seconds 5

# Start Logger Service first (Port 3005)
Start-Service -ServiceName "Logger Service" -Directory "D:\Appointy_Projects\Notli\logger" -Command "go run main.go"

# Wait for Logger Service to initialize
Write-Host "Waiting for Logger Service to initialize..." -ForegroundColor Yellow
Start-Sleep -Seconds 3

# Start Delivery Service (MS4) - Port 3003
Start-Service -ServiceName "Delivery Service" -Directory "D:\Appointy_Projects\Notli\microservice4_delivery" -Command "go run cmd/server/main.go"

Write-Host "All services started! The system is now ready for integration testing." -ForegroundColor Cyan
Write-Host "To run the integration test, execute: cd D:\Appointy_Projects\Notli\tests\integration; go test -v" -ForegroundColor Yellow
Write-Host "To send a test email with UI: Open browser and navigate to http://localhost:3000/static/send_email.html" -ForegroundColor Magenta
Write-Host "Unified logs are available at: http://localhost:3005/logs" -ForegroundColor Magenta
