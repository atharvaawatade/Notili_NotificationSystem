# Manual Testing Procedure for NOTLI End-to-End Flow

Follow these steps to manually test the end-to-end functionality of the NOTLI system:

## Step 1: Register a User and Get API Key

1. Open your browser and navigate to: http://localhost:3000/static/signup.html
2. Register with the following details:
   - Email: test_user@example.com
   - Password: StrongPassword123!
3. After successful registration, you'll be redirected to the dashboard
4. Create a new API key by entering a name (e.g., "Test Key") and clicking "Create Key"
5. **Important:** Copy the API key shown - you'll only see it once!

## Step 2: Send Test Email Using cURL

Run the following cURL command in your terminal, replacing `YOUR_API_KEY` with the API key you copied:

```bash
curl -X POST http://localhost:3001/v1/email/notify \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_API_KEY" \
  -d '{
    "recipient": "atharvaawatade@gmail.com", 
    "subject": "NOTLI Integration Test", 
    "body": "hii test", 
    "from_name": "NOTLI Test", 
    "idempotency_key": "test-123e4567-e89b-12d3-a456-426614174000"
  }'
```

## Step 3: Check the Email

Check the inbox of atharvaawatade@gmail.com for the test email. It should arrive within a few seconds to a minute.

## Performance Measurement

The typical flow timing breakdown is:
- API Key validation: ~100-300ms
- Message queuing (Kafka): ~50-200ms
- Message prioritization: ~50-200ms
- Template rendering: ~100-500ms
- Email provider API call: ~200-3000ms

Total expected time: 500ms to 5 seconds, with the biggest variable being the external email provider's response time.
