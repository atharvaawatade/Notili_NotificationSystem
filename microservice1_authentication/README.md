# NOTLI - Microservice 1: Authentication & API Gateway

This microservice serves as the primary user-facing entry point for the NOTLI notification system. It handles user registration, login, session management (JWT), API key lifecycle, rate limiting, and acts as a proxy for certain requests (like sending emails via API keys).

## Core Features

*   **User Management:** Secure signup and login using email/password.
*   **Session Management:** JWT-based authentication with access and refresh tokens.
*   **API Key Management:** CRUD operations for user-specific API keys used to authenticate other NOTLI services.
*   **Rate Limiting:** Protects API endpoints using a configurable, in-memory token bucket algorithm (per API key or client IP).
*   **Request Proxying:** Securely proxies requests (e.g., email sending) to downstream services using API key authentication.
*   **Password Security:** Uses `bcrypt` for strong password hashing.
*   **Database Integration:** Uses PostgreSQL with GORM for data persistence and automatic migrations.
*   **Modern Frontend:** Includes HTML/JS/CSS for user interaction (Login, Signup, Dashboard, Send Email pages) featuring a dark theme, toasts, loading indicators, and a placeholder rate limit meter.

## Tech Stack

*   **Language:** Go (Golang) 1.21+
*   **Web Framework:** Gin (`github.com/gin-gonic/gin`)
*   **Database:** PostgreSQL
*   **ORM:** GORM (`gorm.io/gorm`, `gorm.io/driver/postgres`)
*   **Password Hashing:** `golang.org/x/crypto/bcrypt`
*   **JWT:** `github.com/golang-jwt/jwt/v5`
*   **Rate Limiting:** `golang.org/x/time/rate` (in-memory)
*   **Configuration:** `.env` file (`github.com/joho/godotenv`)
*   **UUIDs:** `github.com/google/uuid`
*   **Frontend:** HTML, Tailwind CSS (via CDN in examples, potentially local build), Vanilla JavaScript

## Project Structure

```
.env                   # Environment variables (DO NOT COMMIT)
README.md              # This file
go.mod / go.sum       # Go module files
cmd/
  server/
    main.go          # Application entry point, Gin setup, routing, middleware
internal/
  config/            # Configuration loading (config.go)
  database/          # Database connection (database.go)
  handler/           # HTTP request handlers (auth_handler.go, apikey_handler.go, proxy_handler.go, validate_handler.go)
  middleware/        # Gin middleware (auth_middleware.go, ratelimit_middleware.go)
  models/            # Data models/structs (user.go, apikey.go)
  repository/        # Database interaction logic (user_repository.go, apikey_repository.go)
  service/           # Business logic (auth_service.go, apikey_service.go, session_manager.go)
web/
  static/            # Static frontend assets
    css/             # CSS files (style.css, etc.)
    js/              # Frontend JavaScript (auth.js, dashboard.js, send_email.js, etc.)
    signup.html      # Signup page
    login.html       # Login page
    dashboard.html   # User dashboard (API key management)
    send_email.html  # Page to send emails via API key
userintegration.readme # Guide for users integrating with the API key
```

## Setup and Configuration

1.  **Prerequisites:**
    *   Go 1.21 or later installed.
    *   PostgreSQL server running.

2.  **Database Setup:**
    *   Create a PostgreSQL database (e.g., `appointy` or `notli_auth_db`).
    *   Ensure you have a user with privileges on this database.

3.  **Environment Variables:**
    *   Create a `.env` file in the `microservice1_authentication` root directory.
    *   Copy the template below and fill in your specific values.

    ```dotenv
    # .env example

    # Database Configuration
    DB_HOST=localhost
    DB_PORT=5432
    DB_NAME=appointy
    DB_USER=your_db_user
    DB_PASSWORD=your_db_password
    DB_SSL_MODE=disable # Set to 'require' or other appropriate value for production

    # Server Configuration
    SERVER_PORT=3000
    SERVER_HOST=localhost # Or 0.0.0.0 to listen on all interfaces

    # Security
    # Generate a strong, random secret for JWT signing (e.g., openssl rand -hex 32)
    JWT_SECRET=YOUR_VERY_STRONG_RANDOM_JWT_SECRET
    JWT_ACCESS_EXPIRY_MINUTES=15  # Access token expiry time
    JWT_REFRESH_EXPIRY_DAYS=7     # Refresh token expiry time

    # Rate Limiting (Requests per second, applied per API Key or IP)
    RATE_LIMIT_PER_SEC=10 # Default requests per second allowed
    RATE_LIMIT_BURST=20   # Default burst allowance

    # Session Management Note: JWT tokens are stored client-side (e.g., cookies/local storage). 
    # No server-side session store (like Redis) is currently implemented based on code review.
    # SESSION_DRIVER=cookie # Example if cookie-based sessions were used
    # If using Redis for other purposes (e.g., future session store, caching):
    # REDIS_HOST=localhost
    # REDIS_PORT=6379
    # REDIS_PASSWORD=
    # REDIS_DB=1 
    ```

4.  **Install Dependencies:**
    *   Navigate to the `microservice1_authentication` directory.
    *   Run: `go mod tidy`

## Running the Service

1.  **Navigate to the `cmd/server` directory:**
    ```bash
    cd cmd/server
    ```
2.  **Run the application:**
    ```bash
    go run main.go
    ```
    *   The server will start, connect to the database, run migrations (if applicable), and listen on the configured host and port (e.g., `http://localhost:3000`).

3.  **Access the application:**
    *   Open your web browser and go to `http://localhost:3000` (or the host/port you configured). You should be redirected to the login page (`/static/login.html`).

## API Endpoints

*   **Static Files:**
    *   `GET /`: Redirects to `/static/login.html`.
    *   `GET /static/*filepath`: Serves static frontend files (HTML, CSS, JS).
*   **Health Check:**
    *   `GET /health`: Returns `{"status": "UP"}`.
*   **Authentication (Public):**
    *   `POST /api/auth/signup`: Register a new user. Body: `{"email": "...", "password": "..."}`.
    *   `POST /api/auth/login`: Log in a user. Body: `{"email": "...", "password": "..."}`. Returns JWT access/refresh tokens in cookies/body.
    *   `POST /api/auth/refresh`: Obtain a new access token using a valid refresh token (sent via cookie or header).
    *   `POST /api/auth/logout`: Invalidate the refresh token (may involve client-side deletion and/or server-side blocklist if implemented).
    *   `POST /api/auth/validate`: Validate an API key (used internally by other services or for external checks). Body: `{"apiKey": "..."}`. Returns `{"valid": true/false}`.
*   **Email Proxy (Requires API Key Auth):**
    *   `POST /api/proxy/email`: Sends an email request to the message queue service (M2). Requires `X-API-Key` header. Body format specified by M2 (e.g., `{"recipient": "...", "subject": "...", "body": "...", "type": "transactional/promotional"}`).
*   **API Key Management (Requires JWT Auth):**
    *   Protected by `AuthMiddleware` (requires valid JWT access token in `Authorization: Bearer <token>` header or cookie).
    *   `POST /api/keys`: Create a new API key. Body: `{"name": "..."}`. Returns the full API key **only once** upon creation.
    *   `GET /api/keys`: List all API keys (names, prefixes, IDs) for the authenticated user.
    *   `DELETE /api/keys/{keyId}`: Delete a specific API key by its ID.

## Rate Limiting

*   The `RateLimitMiddleware` is applied globally to all routes.
*   It uses an in-memory token bucket approach based on `golang.org/x/time/rate`.
*   Limits are applied **per API Key** (if `X-API-Key` header is present) or **per Client IP Address** otherwise.
*   Configuration is via `.env` variables:
    *   `RATE_LIMIT_PER_SEC`: Tokens added per second (default: 10).
    *   `RATE_LIMIT_BURST`: Maximum tokens in the bucket (default: 20).
*   If the limit is exceeded, the API returns `HTTP 429 Too Many Requests` with a JSON error body.
*   **Performance Consideration:** This in-memory limiter is suitable for single-instance deployments. For multi-instance, distributed rate limiting (e.g., using Redis) would be required.

## Frontend Enhancements

*   **Dark Theme:** Consistent dark mode across all pages (`login`, `signup`, `dashboard`, `send_email`).
*   **Toast Notifications:** Used for success/error feedback (e.g., login success, API key creation, email send status, rate limit errors).
*   **Loading Indicators:** Buttons show spinners and disabled states during API calls (e.g., Login, Signup, Send Email).
*   **Rate Limit Meter:** A visual placeholder progress bar is included on the `send_email.html` page (requires backend integration for live data).
*   **Transitions:** Subtle CSS transitions provide smoother UI interactions.

## Future Considerations

*   Distributed Rate Limiting (Redis).
*   Distributed Session Management/Revocation List (Redis).
*   Password Reset Functionality.
*   Email Verification upon Signup.
*   More robust input validation.
*   Two-Factor Authentication (2FA).
*   Social Logins (OAuth).
*   More detailed user profile management.
*   Admin panel for user/key management.
