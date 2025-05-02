{{ ... }}
# Microservice 4: Delivery Engine

## Overview

The Delivery Engine is the final stage in the NOTLI notification pipeline. It consumes processed and prioritized notification messages from Kafka, renders the appropriate email content using a templating engine, attempts delivery via a configured email provider (primarily Resend), logs the delivery status, and handles retries for transient failures.

## Key Features

*   **Kafka Consumer:** Subscribes to the prioritized notification topic (e.g., `prioritized_notifications` from Microservice 3) using `segmentio/kafka-go`.
*   **Email Delivery:** Sends emails primarily using the Resend API (`resend/resend-go/v2`).
*   **Templating:** Utilizes a templating engine that supports custom HTML templates (prioritizing local green/white themed templates) and potentially Mailjet templates for different email types.
*   **Delivery Logging & Status Tracking:** Records email sending attempts, successes, and failures in a PostgreSQL database (`lib/pq`).
*   **Retry Mechanism:** Implements an exponential backoff retry strategy for emails that fail due to transient issues.
*   **Worker Pool:** Processes messages concurrently using a pool of goroutines.
*   **Health Check:** Provides a `/health` endpoint via Gin.
*   **(Potential) API Endpoints:** May expose additional HTTP endpoints via Gin for tasks like manual email triggers or status checks (needs verification in `internal/handler/email_handler.go`).
*   **Graceful Shutdown:** Handles termination signals to shut down Kafka consumer, database connections, and the HTTP server cleanly.

## Technology Stack

*   **Language:** Go (Golang)
*   **Messaging:** Apache Kafka (Consumer using `github.com/segmentio/kafka-go`)
*   **Email Provider:** Resend (`github.com/resend/resend-go/v2`)
*   **Database:** PostgreSQL (`github.com/lib/pq`) - For logging and retry state.
*   **Web Framework:** Gin (`github.com/gin-gonic/gin`) - Primarily for health checks and potentially other API endpoints.
*   **Templating:** Custom Go template engine (potentially integrating with Mailjet for template storage/retrieval).
*   **Configuration:** `.env` files (`github.com/joho/godotenv`)

## Workflow

1.  Consumes a prioritized message from the input Kafka topic (e.g., `prioritized_notifications`).
2.  Deserializes the message content.
3.  Fetches and renders the appropriate email template (HTML/text) using the templating engine.
4.  Attempts to send the email using the configured Resend provider.
5.  Logs the attempt and status (e.g., `PENDING`, `SENT`, `FAILED`) to the PostgreSQL database.
6.  If sending fails with a retryable error, schedules a retry according to the defined strategy.
7.  Updates the log in PostgreSQL upon successful delivery or terminal failure.

## API Endpoints

*   `GET /health`
    *   **Description:** Basic health check.
    *   **Response:** `200 OK` (likely with status details).
*   **Other Potential Endpoints:** (Verify in `internal/handler/email_handler.go`)
    *   Endpoints related to checking email status or manually triggering emails might exist.

*Note: This service's primary interaction is via Kafka. API endpoints are likely secondary.*

## Setup and Running

1.  **Prerequisites:**
    *   Go (version specified in `go.mod`)
    *   Apache Kafka broker(s) (input topic must exist)
    *   PostgreSQL server
    *   Resend account and API key (with a verified sending domain/email).
    *   (Optional) Mailjet account and API keys if using Mailjet templates.
2.  **Configuration:**
    *   Create a `.env` file in the microservice root directory (`microservice4_delivery`).
    *   Populate it with environment variables (refer to `internal/config/config.go` for required keys):
        ```dotenv
        SERVER_PORT=5001
        
        # Kafka Configuration
        KAFKA_BOOTSTRAP_SERVERS=localhost:9092 # Comma-separated list
        KAFKA_INPUT_TOPIC=prioritized_notifications # Should match M3 output topic
        KAFKA_CONSUMER_GROUP=delivery_group
        # Add other Kafka settings from config.go (timeouts, buffer sizes etc.) if needed

        # Database Configuration
        DB_HOST=localhost
        DB_PORT=5432
        DB_NAME=notli_delivery_db # Or your chosen DB name
        DB_USER=your_db_user
        DB_PASSWORD=your_db_password
        DB_SSL_MODE=disable
        
        # Resend Configuration
        RESEND_API_KEY=re_your_api_key
        RESEND_DOMAIN=yourdomain.com         # Domain you send from
        RESEND_VERIFIED_EMAIL=sender@yourdomain.com # Verified sender email
        RESEND_DETAILED_LOGS=true # Or false
        
        # Mailjet Configuration (Optional - for Templates)
        ENABLE_MAILJET_TEMPLATES=false # Set to true if using Mailjet for templates
        MAILJET_API_KEY=
        MAILJET_SECRET_KEY=
        
        # Email Retry Configuration
        EMAIL_MAX_RETRIES=5
        EMAIL_RETRY_INITIAL_DELAY=5s
        EMAIL_RETRY_MAX_DELAY=1m

        # Add any other required variables from config.go
        ```
3.  **Dependencies:**
    ```bash
    go mod tidy
    ```
4.  **Database Migration:**
    *   The service appears to initialize the database schema on startup via `emailService.InitDB(ctx)`. Ensure the DB user has necessary permissions.
5.  **Run the service:**
    ```bash
    # Option 1: Using go run (for development)
    go run cmd/server/main.go 
    
    # Option 2: Build and run executable (for production)
    # go build -o delivery ./cmd/server
    # ./delivery
    ```

The service will start, connect to Kafka, PostgreSQL, and Resend, begin consuming messages, and expose its health check/API endpoints.

## Environment Variables

Refer to the `.env` example above and `internal/config/config.go` for a complete list and descriptions of required environment variables.