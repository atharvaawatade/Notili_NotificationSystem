# Microservice 2: Message Queue Service

## Overview

The Message Queue Service acts as the primary authenticated entry point for submitting notification requests into the NOTLI system. It receives requests via a REST API, validates the provided API key against the Authentication Service (Microservice 1), ensures request idempotency using a PostgreSQL store, and then publishes the validated notification message onto a Kafka topic for downstream processing.

## Key Features

*   **API Endpoint:** Exposes a single endpoint (`POST /v1/email/notify`) to accept notification requests.
*   **API Key Authentication:** Validates the `X-API-Key` header by communicating with the Authentication Service.
*   **Request Validation:** Validates incoming request bodies using `go-playground/validator` for required fields and correct formats (e.g., email, UUID).
*   **Idempotency:** Prevents duplicate message processing by checking and storing idempotency keys (`UUIDv4`) in a PostgreSQL database.
*   **Kafka Producer:** Publishes successfully validated and non-duplicate notification messages to a configured Kafka topic.
*   **Health Check:** Provides a `/healthz` endpoint for monitoring.
*   **Graceful Shutdown:** Implements graceful shutdown handling for the HTTP server and resource cleanup (Kafka producer, DB connection).

## Technology Stack

*   **Language:** Go (Golang)
*   **Web Framework:** Gin
*   **Messaging:** Apache Kafka (Producer using `github.com/segmentio/kafka-go`)
*   **Database:** PostgreSQL (used for idempotency tracking)
*   **Validation:** `github.com/go-playground/validator/v10`
*   **Configuration:** `.env` files
*   **API Client:** Custom client to interact with the Authentication Service (Microservice 1).

## API Endpoints

*   `POST /v1/email/notify`
    *   **Description:** Submits a notification request for processing.
    *   **Headers:**
        *   `X-API-Key`: (Required) A valid API key obtained from the Authentication Service.
        *   `Content-Type`: `application/json`
    *   **Request Body:** (See `internal/handler/email.go/EmailRequest` struct)
        ```json
        {
          "idempotency_key": "unique-uuid-v4-string", // Required
          "recipient": "recipient@example.com",        // Required, valid email
          "subject": "Your Notification Subject",        // Required
          "body": "The content of your notification.", // Required
          "from_name": "Optional Sender Name",
          "from_email": "optional_sender@example.com", // Optional, valid email
          "reply_to": ["reply-to@example.com"]        // Optional, list of valid emails
        }
        ```
    *   **Responses:**
        *   `202 Accepted`: Request successfully validated and queued.
        *   `400 Bad Request`: Invalid request body or validation failure.
        *   `401 Unauthorized`: Missing `X-API-Key` header.
        *   `403 Forbidden`: Invalid or unauthorized `X-API-Key`.
        *   `409 Conflict`: Duplicate request detected (based on `idempotency_key`).
        *   `500 Internal Server Error`: Failed to validate API key, check idempotency, or enqueue message.
*   `GET /healthz`
    *   **Description:** Health check endpoint.
    *   **Response:** `200 OK` with `{"status": "ok"}`

## Setup and Running

1.  **Prerequisites:**
    *   Go (version specified in `go.mod`)
    *   PostgreSQL server (for idempotency store)
    *   Apache Kafka broker(s)
    *   Running instance of Microservice 1 (Authentication)
2.  **Configuration:**
    *   Create a `.env` file in the microservice root directory (`microservice2_messageq`).
    *   Populate it with environment variables (refer to code for required keys like `SERVER_PORT`, `KAFKA_BROKERS`, `KAFKA_TOPIC`, `AUTH_SERVICE_URL`, `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSL_MODE`):
        ```dotenv
        SERVER_PORT=4000
        
        # Kafka Configuration
        KAFKA_BROKERS=localhost:9092 # Comma-separated list
        KAFKA_TOPIC=notifications
        
        # Authentication Service Dependency
        AUTH_SERVICE_URL=http://localhost:8081 # URL of Microservice 1
        
        # PostgreSQL Idempotency Store Configuration
        DB_HOST=localhost
        DB_PORT=5432
        DB_USER=your_db_user
        DB_PASSWORD=your_db_password
        DB_NAME=notli_messageq_db
        DB_SSL_MODE=disable
        DB_MAX_IDLE_CONNS=5
        DB_MAX_OPEN_CONNS=10
        DB_CONN_MAX_LIFETIME_MINUTES=30
        DB_CONN_MAX_IDLE_TIME_MINUTES=10
        
        # Add any other required variables
        ```
3.  **Dependencies:**
    ```bash
    go mod tidy
    ```
4.  **Database Migration:**
    *   Manually create the idempotency table in your PostgreSQL database (`notli_messageq_db` in the example). Refer to `internal/store/postgres.go` for the required schema (likely a table `idempotency_keys` with columns like `key` (primary key, text) and `created_at` (timestamp)).
5.  **Run the service:**
    ```bash
    go run cmd/server/main.go
    ```

The service should now be running on port 4000 (or as configured) and connected to Kafka and PostgreSQL.

## Environment Variables

Refer to the `.env` example above and the source code (e.g., `internal/config`, `internal/store/postgres.go`, `internal/kafka/producer.go`, `internal/service/auth_client.go`) for a complete list and descriptions of required environment variables.