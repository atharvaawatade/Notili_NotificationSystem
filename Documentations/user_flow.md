# NOTLI: Understanding the Notification Journey

## What is NOTLI?
NOTLI is a powerful notification system that helps businesses send millions of messages to their users quickly and reliably. Think of it as a super-smart post office for digital messages - it can handle emails now, and soon will support text messages, push notifications, and more!

## The Journey of a Notification

### 1. Getting Started 🚀
When a business wants to use NOTLI:
1. They sign up with their credentials (just one click!).
2. On first login, a secure token is issued and stored in the browser for 1 hour to manage their session.
3. If they are a new user, NOTLI creates a default (dummy) profile for them, which they can update later.
4. They get special "keys" (API Keys) to securely send messages.
5. They verify their sending identity (like confirming ownership of their email domain) to ensure messages reach the inbox.
6. They can set up their preferences in a user-friendly dashboard.

### 2. Sending a Message 📧
When a business wants to send a notification:
1. They create their message, *preferably* using pre-made templates managed via the Admin Interface (MS5). (Using templates ensures messages look consistent and professional across all channels, and makes updates much easier!)
2. They specify who should receive it.
3. They choose how important the message is (high priority, normal, or low).
4. They send it through NOTLI's simple REST API to the **Message Queue Service (MS2)**, potentially including personalized details for each recipient.

### 3. Behind the Scenes Magic ✨ (Microservice Flow)

When a notification request hits the **Message Queue Service (MS2)** via the REST API, a sequence of internal operations begins:

#### Step 1: Authentication & Authorization (MS1 via gRPC)
- MS2 makes a synchronous gRPC call to the **Authentication Service (MS1)**.
- MS1 validates the API Key.
- MS1 checks user permissions and rate limits based on their tier.
- If validation fails, an error is returned immediately. If successful, MS1 returns user context (like tier info) to MS2.

#### Step 2: Queuing & Initial Processing (MS2)
- The validated message gets a unique ID (for idempotency).
- MS2 performs initial validation (e.g., basic structure checks).
- Optionally, content moderation checks can be performed here or later.
- The message, enriched with user context from MS1, is published to a specific Kafka topic (e.g., `notifications.pending`).

#### Step 3: Smart Prioritization & Routing (MS3 via Kafka)
- The **Prioritization Service (MS3)** consumes messages from the `notifications.pending` Kafka topic.
- It applies business rules (using its extensible Rule Engine) to determine:
    - **Priority**: Elevating transactional messages over marketing ones.
    - **Rate Limiting**: Ensuring compliance with user tiers and channel limits.
    - **Scheduling**: Delaying non-urgent messages if needed.
    - **Channel Selection**: Potentially refining the target channel based on rules.
- Once processed, MS3 publishes the message to a channel-specific Kafka topic (e.g., `notifications.email.ready`, `notifications.sms.ready`).

#### Step 4: Delivery Preparation & Execution (MS4 via Kafka)
- The **Delivery Service (MS4)** has dedicated consumers for each channel-specific Kafka topic (e.g., `notifications.email.ready`).
- It fetches the appropriate template (using template engines like Handlebars).
- It renders the final message content with personalized data.
- It uses its plugin architecture to interact with the specific delivery provider (e.g., SendGrid for email, Twilio for SMS).
- MS4 handles provider-specific logic, retries, and error management.
- It publishes delivery status updates (sent, failed, bounced) back to Kafka topics (e.g., `notifications.status.updates`).

#### Step 5: Tracking & Analytics (MS5 via Kafka)
- The **Analytics/Admin Service (MS5)** consumes status updates from `notifications.status.updates` and potentially other topics.
- It aggregates data for real-time dashboards (using tools like Flink and storing results in Elasticsearch).
- It makes analytics available via the Admin UI (React) and potentially webhooks.
- MS5 also consumes events related to user actions (like unsubscribes) to update user preferences (stored in PostgreSQL).

This flow ensures scalability, resilience, and clear separation of concerns between the microservices.

### 4. The User's Dashboard & Session Management
After authentication:
1. The user is redirected to the dashboard, where their profile (name, email) and session expiry are shown.
2. A clear logout button is provided to end their session at any time.
3. If the user is new, their dashboard displays default profile data, which they can update later.
4. If the session expires (after 1 hour), the user is automatically logged out and prompted to sign in again.
5. All sensitive actions require a valid session token.

### 5. The Recipient's Experience 📱
From the recipient's perspective:
1. They receive a professionally formatted message
2. The message arrives quickly (especially for important notifications)
3. They can interact with it (click links, respond, etc.)
4. They have the option to unsubscribe or manage their preferences

## Real-World Example

Let's say an e-commerce site needs to send an order confirmation:

1. **The Trigger**: Customer completes a purchase
2. **The Business**: Uses NOTLI to send a confirmation email
3. **NOTLI's Process**:
   - Validates the request
   - Marks it as high-priority (it's a transaction confirmation)
   - Uses the order confirmation template
   - Fills in the customer's details and order information
   - Sends it through the fastest available route
4. **The Customer**: Receives a professional, branded email confirmation within seconds

## Key Benefits

### For Businesses
- Easy to set up and use
- Reliable delivery of messages
- Detailed analytics and tracking
- No technical headaches
- Scales as their business grows

### For Recipients
- Fast delivery of important messages
- Professional, well-formatted notifications
- Consistent experience across all message types
- Control over their notification preferences

## Future Capabilities
NOTLI is expanding to support:
- Text messages (SMS)
- Push notifications for mobile apps
- WhatsApp messages
- Telegram messages
- Voice notifications
- And more!

## Getting Help
Businesses can:
- View real-time status of their notifications
- Access detailed analytics
- Contact support if needed
- Read comprehensive documentation
- Monitor their usage and costs

## Best Practices
1. Use templates for consistent messaging
2. Mark urgent messages as high priority
3. Respect recipient preferences
4. Monitor analytics for insights
5. Test templates before sending to large groups
6. Use appropriate channels for different types of messages

Remember: NOTLI handles all the complex technical details, so businesses can focus on creating great content for their notifications! 