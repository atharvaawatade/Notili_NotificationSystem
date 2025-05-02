M4:
2025/04/27 18:45:30 no messages received from kafka within the allocated time for partition 0 of distributed-notifications at offset 8: [7] Request Timed Out: the request exceeded the user-specified time limit in the request
2025/04/27 18:45:39 no messages received from kafka within the allocated time for partition 0 of distributed-notifications at offset 8: [7] Request Timed Out: the request exceeded the user-specified time limit in the request
2025/04/27 18:45:48 no messages received from kafka within the allocated time for partition 0 of distributed-notifications at offset 8: [7] Request Timed Out: the request exceeded the user-specified time limit in the request
2025/04/27 18:45:57 no messages received from kafka within the allocated time for partition 0 of distributed-notifications at offset 8: [7] Request Timed Out: the request exceeded the user-specified time limit in the request
2025/04/27 18:46:06 no messages received from kafka within the allocated time for partition 0 of distributed-notifications at offset 8: [7] Request Timed Out: the request exceeded the user-specified time limit in the request
2025/04/27 18:46:12 Received Kafka message from topic distributed-notifications, partition: 0, offset: 8
2025/04/27 18:46:12 Raw Kafka message: {"idempotency_key":"3d736b1e-cec1-4137-a571-5ee5622de9dd","recipient":"atharvaawatade@gmail.com","template_id":"","priority":"","message_type":"transactional","channel":"email","channel_data":{"body":"hii test","from_email":"simplivu@simplivu.com","from_name":"NOTLI Test","reply_to":null,"subject":"NOTLI Test Email"},"created_at":"2025-04-27T18:46:12.0428522+05:30","user_tier":"","priority_score":60}
2025/04/27 18:46:12 Processing Kafka message: IdempotencyKey=3d736b1e-cec1-4137-a571-5ee5622de9dd, Channel=email, Recipient=atharvaawatade@gmail.com
2025/04/27 18:46:12 Using custom sender: NOTLI Test <simplivu@simplivu.com>
2025/04/27 18:46:12 Message priority: , priority score: 60
2025/04/27 18:46:12 Sending email to atharvaawatade@gmail.com with subject 'NOTLI Test Email'
2025/04/27 18:46:12 [INFO] Starting email delivery via Resend (ID: resend-1745759772152238100)
2025/04/27 18:46:12 [INFO] Sending email From: Simplivu <simplivu@simplivu.com>, To: [atharvaawatade@gmail.com], Subject: NOTLI Test Email
2025/04/27 18:46:12 [INFO] Sending email via Resend API...
2025/04/27 18:46:12 committed offsets for group email-distribution-service:
        topic: distributed-notifications
                partition 0: 9
2025/04/27 18:46:12 Successfully committed Kafka message at offset 8
2025/04/27 18:46:15 [SUCCESS] Email sent successfully!
2025/04/27 18:46:15 [SUCCESS] Resend message ID: a0e5483f-3d65-4e29-9ffb-4576a6afb589
2025/04/27 18:46:15 Email sent successfully, message ID: a0e5483f-3d65-4e29-9ffb-4576a6afb589
2025/04/27 18:46:21 no messages received from kafka within the allocated time for partition 0 of distributed-notifications at offset 9: [7] Request Timed Out: the request exceeded the user-specified time limit in the request
2025/04/27 18:46:30 no messages received from kafka within the allocated time for partition 0 of distributed-notifications at offset 9: [7] Request Timed Out: the request exceeded the user-specified time limit in the request
2025/04/27 18:46:39 no messages received from kafka within the allocated time for partition 0 of distributed-notifications at offset 9: [7] Request Timed Out: the request exceeded the user-specified time limit in the request
2


M3:
2025/04/27 18:45:11 Redis connection disabled - rate limiting will be bypassed
2025/04/27 18:45:11 Priority service started successfully
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /health                   --> github.com/appointy/notli/microservice3_prioritization/internal/handler.(*HealthHandler).HealthCheck-fm (3 handlers)
[GIN-debug] GET    /readiness                --> github.com/appointy/notli/microservice3_prioritization/internal/handler.(*HealthHandler).ReadinessCheck-fm (3 handlers)
2025/04/27 18:45:11 Starting server on port 3002...
2025/04/27 18:46:12 Rate limiting disabled - processing message for atharvaawatade@gmail.com (type: transactional)



M2:
2025/04/27 18:45:09 Successfully connected to PostgreSQL!
2025/04/27 18:45:09 Table 'idempotency_keys' checked/created successfully.
2025/04/27 18:45:09 Kafka Producer initialized for topic 'prioritized-notifications' on brokers: [localhost:9092]
[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /healthz                  --> main.main.func3 (3 handlers)
[GIN-debug] POST   /v1/email/notify          --> github.com/appointy/notli/microservice2_messageq/internal/handler.(*EmailHandler).Notify-fm (3 handlers)
2025/04/27 18:45:09 Starting server on port 3001...
Using custom sender email: simplivu@simplivu.com
[GIN] 2025/04/27 - 18:46:12 | 202 |    1.4069864s |             ::1 | POST     "/v1/email/notify"


M1:
lease check https://pkg.go.dev/github.com/gin-gonic/gin#readme-don-t-trust-all-proxies for details.
[GIN-debug] Listening and serving HTTP on localhost:3000
[GIN] 2025/04/27 - 18:45:46 | 404 |      1.1302ms |       127.0.0.1 | GET      "/static/css/style.css"
[GIN] 2025/04/27 - 18:45:49 | 404 |            0s |       127.0.0.1 | GET      "/static/css/style.css"
[GIN] 2025/04/27 - 18:45:56 | 201 |    284.1304ms |       127.0.0.1 | POST     "/api/auth/signup"
[GIN] 2025/04/27 - 18:45:56 | 304 |       551.4µs |       127.0.0.1 | GET      "/static/dashboard.html"
[GIN] 2025/04/27 - 18:45:56 | 404 |            0s |       127.0.0.1 | GET      "/static/css/style.css"
[GIN] 2025/04/27 - 18:45:56 | 200 |      4.4147ms |       127.0.0.1 | GET      "/api/keys/"
[GIN-debug] redirecting request 307: /api/keys/ --> /api/keys/
[GIN] 2025/04/27 - 18:46:00 | 201 |    139.9739ms |       127.0.0.1 | POST     "/api/keys/"
[GIN-debug] redirecting request 301: /api/keys/ --> /api/keys/
[GIN] 2025/04/27 - 18:46:00 | 200 |         817µs |       127.0.0.1 | GET      "/api/keys/"
[GIN] 2025/04/27 - 18:46:04 | 404 |            0s |       127.0.0.1 | GET      "/static/css/style.css"
[GIN] 2025/04/27 - 18:46:04 | 304 |       3.759ms |       127.0.0.1 | GET      "/static/js/api_key_poller.js"
[GIN] 2025/04/27 - 18:46:04 | 304 |       565.3µs |       127.0.0.1 | GET      "/static/js/send_email.js"
[GIN] 2025/04/27 - 18:46:04 | 200 |       514.5µs |       127.0.0.1 | GET      "/api/keys/"
[GIN] 2025/04/27 - 18:46:04 | 200 |       528.1µs |       127.0.0.1 | GET      "/api/keys/"
[GIN] 2025/04/27 - 18:46:09 | 200 |      4.9061ms |       127.0.0.1 | GET      "/api/keys/"
2025/04/27 18:46:10 Received email proxy request
2025/04/27 18:46:10 Request body: {"recipient":"atharvaawatade@gmail.com","subject":"NOTLI Test Email","body":"hii test","from_name":"NOTLI Test","from_email":"simplivu@simplivu.com","idempotency_key":"3d736b1e-cec1-4137-a571-5ee5622de9dd"}
[GIN] 2025/04/27 - 18:46:10 | 200 |    267.2872ms |       127.0.0.1 | POST     "/api/auth/validate"
2025/04/27 18:46:12 Response from Message Queue service (status 202): {"idempotency_key":"3d736b1e-cec1-4137-a571-5ee5622de9dd","message":"Notification accepted for processing"}
[GIN] 2025/04/27 - 18:46:12 | 202 |    1.4334825s |       127.0.0.1 | POST     "/api/proxy/email"
[GIN] 2025/04/27 - 18:46:14 | 200 |      3.5486ms |       127.0.0.1 | GET      "/api/keys/"
[GIN] 2025/04/27 - 18:46:20 | 200 |      1.8653ms |       127.0.0.1 | GET      "/api/keys/"