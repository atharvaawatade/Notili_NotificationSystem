# NOTLI K6 Load Testing Suite

This folder contains a comprehensive load testing suite for NOTLI microservices using K6, an open-source load testing tool from Grafana Labs.

## Prerequisites

1. Install K6 from https://k6.io/docs/get-started/installation/
2. Ensure all NOTLI microservices are running

## Directory Structure

```
k6-load-tests/
├── scripts/
│   ├── config.js             # Shared configuration
│   ├── utils.js              # Shared utilities and metrics
│   ├── run-all-tests.js      # Main entry point script
│   ├── ms1-auth/             # Authentication Service (MS1) tests
│   ├── ms2-queue/            # Message Queue Service (MS2) tests
│   ├── ms3-priority/         # Prioritization Engine (MS3) tests
│   └── ms4-delivery/         # Delivery Engine (MS4) tests
└── results/                  # Test results will be stored here
```

## Running Tests

Use the provided PowerShell script to run tests:

```powershell
.\run-k6-tests.ps1 -Service ms2 -Profile stress
```

Parameters:
- `-Service`: Microservice to test (ms1, ms2, ms3, ms4, or all)
- `-Profile`: Test profile (smoke, load, stress, spike, soak)

## Test Profiles

1. **Smoke Test**: Quick verification with minimal load
2. **Load Test**: Simulates normal traffic (100 VUs)
3. **Stress Test**: Progressively increases load to 1200 VUs
4. **Spike Test**: Sudden traffic surge to 1500 VUs
5. **Soak Test**: Extended load test (3 hours with 400 VUs)

## Microservices

### MS1 - Authentication Service
- Tests user signup, login, API key generation and validation
- Primary endpoint: http://localhost:8081

### MS2 - Message Queue Service
- Tests notification submission, batch processing, and idempotency
- Primary endpoint: http://localhost:8082

### MS3 - Prioritization Engine
- Tests priority rules, message simulation, and queue status
- Primary endpoint: http://localhost:8083

### MS4 - Delivery Engine
- Tests email rendering, templates, and delivery simulation using test providers
- Primary endpoint: http://localhost:8084

## Key Features

- **Realistic Traffic**: Simulates real-world usage patterns
- **Comprehensive Metrics**: Response time, throughput, error rates
- **Test Modes**: All tests use test providers (no real emails sent)
- **Configurable**: Easily adjust VUs, duration, and thresholds

## Advanced Usage

### Custom Environment Variables

```
k6 run scripts/run-all-tests.js -e SERVICE=ms2 -e PROFILE=stress -e ENV=staging
```

### Saving Results to JSON

```
k6 run scripts/ms2-queue/queue-load-test.js --out json=results/ms2-results.json
```

### Running with Grafana Cloud

If you have a Grafana Cloud account, add your API token:

```
k6 run scripts/run-all-tests.js -o cloud
```

## Current Microservice Breaking Points

Based on our optimization work:

- MS1 (Auth): ~350 concurrent users
- MS2 (Queue): ~1200 concurrent users (previously 280)
- MS3 (Priority): ~420 concurrent users
- MS4 (Delivery): ~310 concurrent users
