# Microservice 3: Prioritization Engine

## Overview

The Prioritization Engine is a background processing service within the NOTLI system. Its primary responsibility is to consume notification messages from an input Kafka topic (populated by the Message Queue Service), apply business logic for prioritization and rate limiting, and then produce the processed messages onto an output Kafka topic for consumption by the Delivery Engine (Microservice 4).

## Key Features

*   **Kafka Consumer:** Reads notification messages from a specified input Kafka topic.
*   **Prioritization Logic:** Applies internal rules (via `priority.RuleEngine`) to determine the importance or order of messages.
*   **Rate Limiting:** Enforces rate limits (e.g., for transactional vs. promotional messages) using Redis (`go-redis/redis/v8`) to manage state.
*   **Kafka Producer:** Publishes the processed and prioritized messages to a specified output Kafka topic.
*   **Background Processing:** Runs as a continuous service processing messages from the queue.
*   **Health Check:** Exposes a basic HTTP health check endpoint using Gin.
*   **Graceful Shutdown:** Handles SIGINT/SIGTERM signals for orderly shutdown of Kafka connections, Redis client, and the HTTP server.

## Technology Stack

*   **Language:** Go (Golang)
*   **Messaging:** Apache Kafka (Consumer & Producer using `github.com/segmentio/kafka-go`)
*   **Rate Limiting Cache:** Redis (`github.com/go-redis/redis/v8`)
*   **Web Framework:** Gin (used only for health check endpoint)
*   **Configuration:** `.env` files (or environment variables)

## Workflow

1.  Consumes a message from the input Kafka topic (e.g., `notifications`).
2.  Deserializes the message.
3.  Applies prioritization rules using the `RuleEngine`.
4.  Checks and applies rate limits using the `RateLimiter` (backed by Redis).
5.  If the message passes rate limits and prioritization, it's potentially modified (e.g., with priority metadata).
6.  Produces the processed message to the output Kafka topic (e.g., `prioritized_notifications`).

## API Endpoints

*   `GET /health` (or similar, check `internal/handler/health_handler.go`)
    *   **Description:** Basic health check endpoint.
    *   **Response:** `200 OK` (likely with a simple status message).

*Note: This service primarily interacts via Kafka topics and does not expose a significant external API beyond the health check.*

## Setup and Running

1.  **Prerequisites:**
    *   Go (version specified in `go.mod`)
    *   Apache Kafka broker(s)
    *   Redis server
2.  **Configuration:**
    *   Create a `.env` file in the microservice root directory (`microservice3_prioritization`).
    *   Populate it with environment variables (refer to `internal/config/config.go` for required keys):
        ```dotenv
        SERVER_PORT=5000
        
        # Kafka Configuration
        KAFKA_BOOTSTRAP_SERVERS=localhost:9092 # Comma-separated list
        KAFKA_INPUT_TOPIC=notifications
        KAFKA_OUTPUT_TOPIC=prioritized_notifications
        KAFKA_CONSUMER_GROUP=prioritization_group
        
        # Redis Configuration (for Rate Limiting)
        REDIS_HOST=localhost
        REDIS_PORT=6379
        REDIS_PASSWORD=
        REDIS_DB=0 
        
        # Rate Limiting Rules (Example: requests per second)
        RATE_LIMIT_TRANSACTIONAL=100
        RATE_LIMIT_PROMOTIONAL=10
        
        # Add any other required variables
        ```
3.  **Dependencies:**
    ```bash
    go mod tidy
    ```
4.  **Run the service:**
    ```bash
    # Option 1: Using go run (for development)
    go run cmd/server/main.go 
    
    # Option 2: Build and run executable (for production)
    # go build -o processor ./cmd/server
    # ./processor
    ```

The service will start, connect to Kafka and Redis, begin consuming messages from the input topic, and expose the health check endpoint.

## Environment Variables

Refer to the `.env` example above and `internal/config/config.go` for a complete list and descriptions of required environment variables.