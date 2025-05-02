// Message Queue Service (MS2) Load Test
import { sleep, check } from 'k6';
import http from 'k6/http';
import { config } from '../config.js';
import { metrics, generators, helpers } from '../utils.js';

// Default test configuration - can be overridden via environment variables
export const options = {
  // Use the stress test profile by default
  stages: config.profiles.stress.stages,
  thresholds: {
    ...config.thresholds,
    // Additional service-specific thresholds
    'queue_latency': ['p(95)<300'], // 95% of queue requests under 300ms
  },
};

// Submit a single notification test
export function testSubmitNotification(apiKey) {
  const idempotencyKey = generators.uuid();
  
  const payload = JSON.stringify({
    idempotency_key: idempotencyKey,
    recipient: generators.randomEmail(),
    subject: `Test Subject ${generators.randomString(8)}`,
    body: `This is a test email body generated for load testing. Random ID: ${generators.randomString(16)}`,
    from_name: 'NOTLI Load Test',
    reply_to: ['noreply@notli.example.com']
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': apiKey
    },
  };
  
  const res = http.post(`${config.urls.queue}/v1/email/notify`, payload, params);
  helpers.handleResponse(res, 'queue');
  
  return { idempotencyKey, res };
}

// Test idempotency with duplicate requests
export function testIdempotency(apiKey, idempotencyKey) {
  const payload = JSON.stringify({
    idempotency_key: idempotencyKey,
    recipient: generators.randomEmail(),
    subject: 'Duplicate Request Test',
    body: 'This is a duplicate request with the same idempotency key',
    from_name: 'NOTLI Load Test',
    reply_to: ['noreply@notli.example.com']
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': apiKey
    },
  };
  
  const res = http.post(`${config.urls.queue}/v1/email/notify`, payload, params);
  
  // For idempotency test, we expect a 409 Conflict response
  check(res, {
    'idempotency check passed': (r) => r.status === 409,
  });
  
  helpers.handleResponse(res, 'queue');
  
  return res;
}

// Test batch notification submission (high volume)
export function testBatchNotification(apiKey, count = 10) {
  const requests = [];
  
  for (let i = 0; i < count; i++) {
    requests.push({
      idempotency_key: generators.uuid(),
      recipient: generators.randomEmail(),
      subject: `Batch Test ${i}`,
      body: `This is test email ${i} in a batch of ${count}. Random text: ${generators.randomString(20)}`,
      from_name: 'NOTLI Batch Test',
      reply_to: ['noreply@notli.example.com']
    });
  }
  
  const payload = JSON.stringify(requests);
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': apiKey
    },
  };
  
  const res = http.post(`${config.urls.queue}/v1/email/notify-batch`, payload, params);
  helpers.handleResponse(res, 'queue');
  
  return res;
}

// Test mixed message types
export function testMixedTypes(apiKey) {
  // Here we would test different types of notifications if supported
  // For now we'll just use a different subject/body format
  const payload = JSON.stringify({
    idempotency_key: generators.uuid(),
    recipient: generators.randomEmail(),
    subject: 'URGENT: Mixed Type Test',
    body: '<h1>HTML Test</h1><p>This is a test with HTML content to test different message types.</p>',
    from_name: 'NOTLI Priority Test',
    reply_to: ['urgent@notli.example.com']
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': apiKey
    },
  };
  
  const res = http.post(`${config.urls.queue}/v1/email/notify`, payload, params);
  helpers.handleResponse(res, 'queue');
  
  return res;
}

// Main test function
export default function() {
  // Get API key from previous run or fallback to default
  const apiKey = helpers.getApiKey() || 'test-api-key-12345-abcdef';
  
  // 1. Submit a notification
  const { idempotencyKey, res } = testSubmitNotification(apiKey);
  
  // If successful, test idempotency with the same key
  if (res.status === 202 || res.status === 200) {
    sleep(0.5);
    testIdempotency(apiKey, idempotencyKey);
  }
  
  // 2. Test batch submissions (high volume) less frequently
  if (Math.random() < 0.3) { // 30% chance
    sleep(1);
    testBatchNotification(apiKey, Math.floor(Math.random() * 20) + 5); // 5-25 items
  }
  
  // 3. Test mixed message types occasionally
  if (Math.random() < 0.2) { // 20% chance
    sleep(0.5);
    testMixedTypes(apiKey);
  }
  
  // Random sleep between requests to simulate user behavior
  sleep(Math.random() * 1.5 + 0.5);
}
