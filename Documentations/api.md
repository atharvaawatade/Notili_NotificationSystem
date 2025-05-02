# NOTLI API Documentation

## Table of Contents
1. [Overview](#overview)
2. [API Standards](#api-standards)
3. [Public REST APIs](#public-rest-apis)
    * [Authentication (MS1)](#authentication-ms1)
    * [Notification Submission (MS2)](#notification-submission-ms2)
    * [Administration (MS5)](#administration-ms5)
4. [Internal Communication Patterns](#internal-communication-patterns)
    * [Synchronous (gRPC)](#synchronous-grpc)
    * [Asynchronous (Kafka)](#asynchronous-kafka)
    * [Visual Flow](#visual-flow)
5. [Webhooks](#webhooks)
6. [Database Schema Overview](#database-schema-overview)
7. [Security Considerations](#security-considerations)
8. [Channel Specific Data (Examples)](#channel-specific-data-examples)

## Overview

NOTLI employs a robust and scalable communication strategy using:

* **Public REST APIs**: Secure `HTTPS/JSON` endpoints for external clients (users/applications) to interact with the system (submit notifications, manage account/API keys).
* **Internal gRPC**: High-performance, low-latency `RPC` calls using Protocol Buffers for synchronous request/response communication *between* microservices.
* **Asynchronous Kafka**: A resilient event backbone using Apache Kafka for decoupling microservices, buffering messages, and enabling stream processing. Messages are typically serialized using Protocol Buffers.

This hybrid approach optimizes for both external accessibility and internal efficiency and resilience.

## API Standards

Consistency is key for a seamless developer experience.

### Public REST API Standards

* **Base URL**: `https://api.notli.com/v1` (Production)
* **Authentication**: Primarily via `X-API-Key` header for notification submission. Bearer Token (JWT obtained via OAuth) for user management and API key generation.
* **Content-Type**: `application/json` for request and response bodies.
* **Rate Limiting**: Enforced per API key, potentially adaptive. Limits returned via headers (e.g., `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`).
* **Versioning**: Via URL path (`/v1/`, `/v2/`).
* **Idempotency**: Supported via `Idempotency-Key` header for critical POST requests (like notification submission) to prevent accidental duplicates.
* **Error Format**: Standardized JSON structure (see below).

### Internal gRPC Standards

* **Purpose**: Used for essential, low-latency internal requests where an immediate response is required (e.g., API key validation).
* **Schema**: Service definitions (`.proto` files) using Protocol Buffers v3 define the contracts.
* **Transport**: Secure communication via mutual TLS (mTLS).
* **Service Discovery**: Relies on the underlying orchestrator (e.g., Kubernetes Services) for locating service instances.
* **Load Balancing**: Handled by the infrastructure (e.g., Kubernetes service mesh like Istio/Linkerd or native gRPC load balancing).
* **Error Handling**: Standard gRPC status codes and error details.

### Asynchronous Kafka Standards

* **Purpose**: Decouples microservices, handles buffering for high throughput, enables event-driven processing and analytics.
* **Serialization**: Protocol Buffers v3 used for message schemas to ensure type safety and efficiency.
* **Topics**: Clearly named topics represent specific event types or stages in the notification lifecycle (e.g., `prioritized-notifications`, `delivery-status-events`).
* **Partitioning**: Strategies vary by topic (e.g., by `user_id` for transactional order, round-robin for promotional throughput).
* **Schema Registry**: Recommended for managing Protobuf schema evolution (e.g., Confluent Schema Registry).

### Common Error Response Format (REST)

```json
{
  "error": {
    "code": "INVALID_ARGUMENT", // Machine-readable error code (e.g., AUTH_FAILURE, RATE_LIMIT_EXCEEDED, VALIDATION_ERROR)
    "message": "Invalid recipient email address provided.", // Human-readable description
    "target": "personalizations[0].to[0].email", // Optional: Field related to the error
    "details": [ // Optional: More specific error details if applicable
      {"field": "email", "reason": "Address does not contain '@' symbol"}
    ]
  },
  "request_id": "uuid-for-tracking" // Unique ID for this specific request
}
```

## Public REST APIs

These are the primary endpoints exposed to external clients.

### Authentication (MS1)

*Microservice responsible: `microservice1_authentication` (Go/Gin)*

Handles user onboarding, login, JWT session management, API key lifecycle management, rate limiting, and proxies certain authenticated requests (like email submission).

```http
# --- Authentication --- 

# Register New User
POST /api/auth/signup
Content-Type: application/json

Request:
{"email": "user@example.com", "password": "your_password"}

Response 201 Created: (Success message, no body usually)

# Login User
POST /api/auth/login
Content-Type: application/json

Request:
{"email": "user@example.com", "password": "your_password"}

Response 200 OK:
(JWT Access/Refresh tokens likely returned in HttpOnly Cookies and/or JSON body)
{
  "message": "Login successful",
  "access_token": "ey...", // Optional if using cookies primarily
  "refresh_token": "ey..." // Optional if using cookies primarily
}

# Refresh JWT Access Token
POST /api/auth/refresh
(Requires valid Refresh Token, usually sent via Cookie)

Response 200 OK:
(New JWT Access token likely returned in HttpOnly Cookie and/or JSON body)
{
  "access_token": "ey..." // Optional if using cookies primarily
}

# Logout User
POST /api/auth/logout
(Requires valid Refresh Token, usually sent via Cookie - invalidates refresh token server-side if implemented)

Response 200 OK: (Success message)

# --- API Key Management (Requires JWT Auth) --- 
# (Authorization: Bearer <user_jwt_access_token> or via Cookie)

# Generate API Key
POST /api/keys
Content-Type: application/json

Request:
{
  "name": "My Production Email Key" // User-friendly name for the key
}

Response 201 Created:
{
  "api_key": "ntlk_prod_xxxxxxxxxxxx", // The generated key (show only once!)
  "key_id": "uuid-key-id", 
  "name": "My Production Email Key",
  "prefix": "ntlk_prod_",
  "created_at": "timestamp"
}

# List API Keys
GET /api/keys

Response 200 OK:
{
  "api_keys": [
    {
      "key_id": "uuid-key-id-1",
      "name": "My Production Email Key",
      "prefix": "ntlk_prod_",
      "created_at": "timestamp"
      // Note: The actual key value is NOT returned here
    }
    // ... other keys
  ]
}

# Delete API Key
DELETE /api/keys/{keyId}

Response 204 No Content

# --- Internal/Service Endpoints --- 

# Validate API Key (Used by MS2)
POST /api/auth/validate 
Content-Type: application/json

Request:
{"apiKey": "ntlk_prod_xxxxxxxxxxxx"}

Response 200 OK:
{"valid": true} or {"valid": false}

# Proxy Email Request (Used by external clients via API Key)
POST /api/proxy/email
X-API-Key: <your_notli_api_key>
Content-Type: application/json

Request: (Body format expected by MS2)
{
  "recipient": "target@example.com", 
  "subject": "Your Subject Here", 
  "body": "HTML or text body", 
  "type": "transactional" // or "promotional"
}

Response: (Proxied response from MS2, e.g., 202 Accepted)

# Health Check
GET /health
Response 200 OK:
{"status": "UP"}

```

### Notification Submission (MS2)

*Microservice responsible: `microservice2_messageq` (Go/Gin)*

Receives notification requests, validates the API key (via call to MS1), checks idempotency (using PostgreSQL), and publishes valid messages to the primary Kafka topic (`notifications`).

```http
POST /api/notify
Content-Type: application/json
X-API-Key: <your_notli_api_key> 
Idempotency-Key: <unique-request-identifier-uuid> // Optional but recommended

Request Body:
{
  "recipient": "recipient@example.com", // Required
  "subject": "Your Subject", // Required
  "body": "<p>HTML content</p> or Plain text content", // Required
  "type": "transactional", // Required: "transactional" or "promotional"
  "sender_email": "sender@yourdomain.com", // Optional: If different from a default
  "sender_name": "Your App Name" // Optional
  // Other potential fields like template_id, custom_args etc. might be supported
}

Response 202 Accepted:
{
  "message": "Notification request accepted for processing.",
  "message_id": "uuid-generated-by-ms2" // ID for tracking within Kafka/system
}

Response 4xx/5xx: (Standard Error Format - see above)
  - 400 Bad Request (Invalid input, missing fields)
  - 401 Unauthorized (Invalid or missing API Key)
  - 409 Conflict (Idempotency key reuse with different payload)
  - 429 Too Many Requests (Rate limit exceeded - handled by MS1)
  - 500 Internal Server Error

# Health Check
GET /health
Response 200 OK: (JSON status)

```

### Prioritization Engine (MS3)

*Microservice responsible: `microservice3_prioritization` (Go/Gin)*

This service primarily interacts **asynchronously via Kafka**. It consumes messages from the `notifications` topic, applies prioritization logic and rate limiting (using Redis), and produces messages to the `prioritized_notifications` topic.

It typically does **not** expose significant public REST APIs beyond a health check.

```http
# Health Check
GET /health 
Response 200 OK: (JSON status)
```

### Delivery Engine (MS4)

*Microservice responsible: `microservice4_delivery` (Go/Gin)*

This service primarily interacts **asynchronously via Kafka**. It consumes messages from the `prioritized_notifications` topic, renders templates, sends emails via the configured provider (Resend), and logs status/retries to PostgreSQL.

It may expose a health check and potentially internal API endpoints for status checking or manual triggers.

```http
# Health Check
GET /health 
Response 200 OK: (JSON status)

# Potential Internal Endpoints (Example - Verify implementation)
# GET /api/emails/{message_id}/status 
# POST /api/emails/send 
```

### Administration (MS5)

*Microservice responsible: `microservice5_analytics` (Assumed Go/React based on Arch doc)*

Provides administrative oversight, analytics dashboards, and system management capabilities. Accessed via a web UI, which interacts with these backend APIs.

**Authentication**: Requires administrative user credentials (likely managed via MS1 OAuth/JWT with specific admin roles/scopes).

```http
# --- Analytics --- #

# Get Notification Statistics Overview
GET /admin/stats/overview?time_range=24h
Authorization: Bearer <admin_jwt>

Response 200 OK:
{
  "total_received": 1500000,
  "total_processed": 1495000,
  "total_sent": 1480000,
  "total_delivered": 1450000,
  "total_failed": 15000,
  "average_latency_ms": 250,
  "channels": {
    "email": { "sent": 1000000, "failed": 10000 },
    "sms": { "sent": 480000, "failed": 5000 }
  }
}

# Get Detailed Notification History (with filtering & pagination)
GET /admin/notifications?status=failed&channel=email&limit=50&offset=0
Authorization: Bearer <admin_jwt>

Response 200 OK:
{
  "notifications": [
    { "id": "uuid", "recipient": "...", "status": "failed", "error": "...", "timestamp": "..." },
    // ...
  ],
  "total_count": 15000,
  "limit": 50,
  "offset": 0
}

# --- System Management --- #

# Get System Health / Microservice Status
GET /admin/health
Authorization: Bearer <admin_jwt>

Response 200 OK:
{
  "overall_status": "healthy",
  "services": [
    { "name": "ms1-auth", "status": "healthy", "instances": 3 },
    { "name": "ms2-messageq", "status": "healthy", "instances": 5 },
    // ...
  ]
}

# Get Configuration Overview (Read-Only)
GET /admin/config
Authorization: Bearer <admin_jwt>

# Manage Prioritization Rules (if dynamic rules are implemented)
GET /admin/rules/prioritization
POST /admin/rules/prioritization
PUT /admin/rules/prioritization/{rule_id}
DELETE /admin/rules/prioritization/{rule_id}
Authorization: Bearer <admin_jwt>

# --- User/Account Management (Potentially via MS1 gRPC) --- #

# List Users/Accounts
GET /admin/users
Authorization: Bearer <admin_jwt>

# View/Manage User Details
GET /admin/users/{user_id}
Authorization: Bearer <admin_jwt>

# Manage API Keys (Potentially via MS1 gRPC)
GET /admin/api-keys?user_id={user_id}
DELETE /admin/api-keys/{key_id}
Authorization: Bearer <admin_jwt>
```

## Internal Communication Patterns

Microservices communicate internally using gRPC for synchronous requests and Kafka for asynchronous events.

### Synchronous (gRPC)

Used for critical, low-latency internal requests.

*   **MS2 (Message Queue) -> MS1 (Authentication): `ValidateApiKey`**
    *   **Purpose**: When MS2 receives a notification submission request via the REST API, it synchronously calls MS1 to validate the provided `X-API-Key`.
    *   **Request**: `ValidateApiKeyRequest { api_key: string }`
    *   **Response**: `ValidateApiKeyResponse { is_valid: bool, user_id: string, account_id: string, rate_limit_info: RateLimit, permissions: list<string> }` or gRPC error (e.g., `UNAUTHENTICATED`).
    *   **Importance**: Ensures only valid keys can submit notifications and provides necessary context (like rate limits) for MS2.

*   **MS5 (Admin UI Backend) -> MS1 (Authentication): `ManageApiKeys` (Hypothetical)**
    *   **Purpose**: If administrative actions on API keys (beyond user self-service) are needed.
    *   **Request/Response**: Specific gRPC methods for listing, revoking, or updating metadata for keys across users (requires admin privileges).

### Asynchronous (Kafka)

Decouples services, handles load, enables event-driven workflows.

*   **Topic: `raw-notifications`**
    *   **Producer**: MS2 (Message Queue)
    *   **Consumer**: MS3 (Prioritization Engine)
    *   **Payload**: Protobuf serialization of the validated incoming notification request (structure similar to the `/notifications` REST API body), including metadata like `user_id`, `account_id`, `received_timestamp`.
    *   **Purpose**: Queues validated notifications for prioritization.
    *   **Partitioning**: Likely by `account_id` or `user_id` to maintain per-customer order if needed, or round-robin for general distribution.

*   **Topic: `prioritized-notifications-{channel}` (e.g., `prioritized-notifications-email`, `prioritized-notifications-sms`)**
    *   **Producer**: MS3 (Prioritization Engine)
    *   **Consumer**: MS4 (Delivery Engine) - Specific instances/consumers might subscribe based on the channel.
    *   **Payload**: Protobuf serialization of the notification data, potentially augmented with priority score, calculated rate limits, target delivery provider info, and specific channel routing details.
    *   **Purpose**: Routes prioritized messages to the appropriate delivery channel queues.
    *   **Partitioning**: Could be based on priority, target provider, or recipient characteristics.

*   **Topic: `delivery-status-updates`**
    *   **Producer**: MS4 (Delivery Engine)
    *   **Consumer**: MS5 (Analytics & Admin)
    *   **Payload**: Protobuf message detailing the outcome of a delivery attempt. Includes `notification_id`, `recipient_identifier`, `channel`, `status` (e.g., `SENT`, `FAILED`, `BOUNCED`, `DELIVERED`, `OPENED`, `CLICKED`), `timestamp`, `provider_details`, `error_message` (if failed).
    *   **Purpose**: Feeds the analytics pipeline and allows tracking of notification lifecycle events.
    *   **Partitioning**: Often by `notification_id` or `user_id`.

*   **Topic: `system-events` (Example)**
    *   **Producers**: Various Microservices (MS1-MS5)
    *   **Consumers**: MS5 (Analytics), Monitoring Systems
    *   **Payload**: Generic event structure capturing service health (e.g., startup, shutdown, error rates), scaling events, configuration changes, administrative actions.
    *   **Purpose**: Centralized logging and monitoring of system health and activity.

### Visual Flow

Refer to the **[High-Level Diagram (HLD)](hld.mermaid)** for a visual representation of these interactions.

## Webhooks

NOTLI relies on webhooks from delivery providers (e.g., SendGrid, Twilio) for real-time status updates (delivered, bounced, opened, clicked). Microservice 5 (or potentially a dedicated ingestion service) is responsible for receiving these webhooks.

*   **Endpoint Security**: Webhook endpoints exposed by NOTLI must be secured using:
    *   **HTTPS**: Mandatory.
    *   **Signature Verification**: Providers typically sign webhook requests (e.g., using HMAC-SHA256 with a shared secret or public/private keys). NOTLI must verify these signatures upon receipt.
    *   **IP Whitelisting**: Optionally restrict incoming requests to known provider IP ranges.
*   **Processing**: Received webhooks are typically:
    1.  Validated (signature, basic structure).
    2.  Acknowledged quickly to the provider (e.g., HTTP 200 OK).
    3.  Published as standardized internal events onto a dedicated Kafka topic (e.g., `provider-webhook-events`).
    4.  Consumed by MS5 (Analytics) or MS4 (if immediate action based on status is needed) for further processing.

## Database Schema Overview

*(This section provides a conceptual overview. Refer to individual microservice READMEs or schema files for exact definitions.)*

*   **Authentication DB (MS1 - PostgreSQL)**: Stores `users`, `api_keys`, OAuth credentials, potentially basic user preferences.
*   **Notification Metadata DB (MS2 - PostgreSQL)**: Stores initial submission details, `message_id`, `idempotency_key` records (short-term).
*   **Delivery State DB (MS4 - PostgreSQL/NoSQL?)**: May track inflight delivery attempts, provider correlation IDs, retry counts. (Could also rely heavily on Kafka stream state).
*   **Analytics DB (MS5 - Elasticsearch)**: Stores aggregated metrics, indexed event data for querying.
*   **Admin Config DB (MS5 - PostgreSQL/Shared)**: Stores administrative settings, template definitions (if managed here), DLQ message details.
*   **Distributed Cache/State (Redis)**: Used across services for rate limiting counters, short-term state, cached metrics.

## Security Considerations

Security is paramount.

*   **Authentication:** Use strong, unique API keys; rotate regularly. Protect keys as sensitive secrets.
*   **Authorization:** Ensure API keys grant minimum necessary permissions (scope).
*   **Input Validation:** **Crucially, always sanitize and validate any user-provided data** used within notification templates (`dynamic_data`) on *your* server-side *before* sending it to the NOTLI API. This prevents injection attacks (HTML/script injection) if templates render user input.
*   **Transport Security:** Enforce HTTPS for all public REST APIs and mTLS for internal gRPC.
*   **Webhooks:** Secure endpoints using signature verification and HTTPS.
*   **Rate Limiting:** Protects against abuse and denial-of-service.
*   **Dependency Scanning:** Regularly scan dependencies for vulnerabilities.
*   **Secrets Management:** Use a secure vault for storing secrets (API keys, DB passwords, etc.).

## Channel Specific Data (Examples)

While the core `/notifications` API aims for consistency, the `personalizations[*].recipient_identifiers` (future) and `validated_channel_data` (internal Kafka messages) will hold channel-specific details.

*   **Email:** `to`, `cc`, `bcc` arrays with `email` and `name` fields. `from`, `reply_to`. Attachments, tracking settings.
*   **SMS (Future):** Recipient identifier might be `phone` (E.164 format). `sender_id` (alphanumeric or shortcode). Message body limits.
*   **Push (Future):** Recipient identifier `device_token`. Platform (APNS, FCM). Payload structure (title, body, icon, data payload, sound, badge).
*   **WhatsApp (Future):** Recipient identifier `phone`. Template name (HSM). Template parameters.

*(Refer to specific channel provider documentation for detailed field requirements as channels are implemented.)*