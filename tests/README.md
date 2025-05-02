# NOTLI Integration Testing

This directory contains tests and utilities for verifying the integration between all four NOTLI microservices:

1. **Authentication Service (MS1)** - User registration, login, and API key management
2. **Message Queue Service (MS2)** - Notification ingestion and validation
3. **Prioritization Service (MS3)** - Message prioritization and queue management
4. **Delivery Service (MS4)** - Template rendering and email delivery

## Prerequisites

Before running the integration tests, ensure you have:

1. **Go 1.21+** installed
2. **PostgreSQL** running with the database `appointy` created
3. **Kafka** and **Zookeeper** running (if testing with the actual Kafka, not mocks)
4. **Redis** running (used by rate limiting in MS3)

## Environment Setup

Each microservice uses environment variables specified in their respective `.env` files. The configurations have been updated to ensure each service runs on a different port:

- MS1 (Authentication): Port 3000
- MS2 (Message Queue): Port 3001
- MS3 (Prioritization): Port 3002
- MS4 (Delivery): Port 3003

## Running All Services

For convenience, a PowerShell script is provided to start all microservices:

```
.\run_all_services.ps1
```

This script will:
1. Start the Authentication Service (MS1)
2. Wait 5 seconds for it to initialize
3. Start the Message Queue Service (MS2)
4. Wait 5 seconds for it to initialize
5. Start the Prioritization Service (MS3)
6. Wait 5 seconds for it to initialize
7. Start the Delivery Service (MS4)

Each service will run in its own terminal window for easy monitoring.

## Running the Integration Tests

After all services are running, you can execute the integration tests:

```
cd integration
go test -v
```

The end-to-end test will:
1. Register a test user with the Authentication Service
2. Create an API key for the user
3. Use that API key to send a notification through the Message Queue Service
4. Wait for the notification to be processed by the Prioritization Service and delivered by the Delivery Service

A successful test will send a real email to the specified test address (by default: atharvaawatade@gmail.com).

## Customizing the Test

You can customize the test by setting environment variables or modifying the `.env.test` file. Available configurations:

- `TEST_EMAIL_ADDRESS`: The email address to send the test notification to
- `REGISTRATION_EMAIL`: The email to use for registering the test user
- `REGISTRATION_PASSWORD`: The password for the test user
- `AUTH_SERVICE_URL`: URL of the Authentication Service
- `MESSAGE_QUEUE_URL`: URL of the Message Queue Service
- `PRIORITIZATION_URL`: URL of the Prioritization Service
- `DELIVERY_SERVICE_URL`: URL of the Delivery Service

## Manual Testing

If you prefer to test manually, follow these steps:

1. Start all services using the `run_all_services.ps1` script
2. Register a user at http://localhost:3000/static/signup.html
3. Log in at http://localhost:3000/static/login.html
4. Create an API key on the dashboard
5. Use the API key to send a notification:

```bash
curl -X POST http://localhost:3001/v1/email/notify \
  -H "Content-Type: application/json" \
  -H "X-Api-Key: YOUR_API_KEY" \
  -d '{
    "message_type": "transactional",
    "priority": 5,
    "channel": "email",
    "recipient": "your_test_email@example.com",
    "subject": "NOTLI Test Notification",
    "content": {
      "body": "This is a test notification from NOTLI.",
      "footer_text": "If you received this email, the test was successful!"
    },
    "idempotency_key": "manual-test-123"
  }'
```

## Troubleshooting

If the integration tests fail, check:

1. All services are running properly (check terminal windows)
2. Database connection is working (check logs for connection errors)
3. Kafka and Redis are running if required
4. The services are using the correct ports as specified in the `.env` files
5. Network connections between services are not blocked

Each service logs its operations, so check the terminal outputs for specific errors.
