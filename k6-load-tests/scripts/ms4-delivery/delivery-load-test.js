// Delivery Engine (MS4) Load Test - Using TEST PROVIDER (No real emails)
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
    // Service-specific thresholds
    'delivery_latency': ['p(95)<400'], // 95% of delivery requests under 400ms
  },
};

// Test health endpoint
export function testHealth() {
  const res = http.get(`${config.urls.delivery}/health`);
  helpers.handleResponse(res, 'delivery');
  
  return res;
}

// Test email rendering (using test provider ONLY)
export function testEmailRendering(apiKey) {
  // Use a specific test provider flag to ensure no real emails are sent
  const payload = JSON.stringify({
    template_id: 'test-template',
    recipient: generators.randomEmail(),
    subject: `Test Email ${generators.randomString(8)}`,
    data: {
      user_name: 'Test User',
      company_name: 'NOTLI Testing',
      action_url: 'https://example.com/test',
      current_date: new Date().toISOString().split('T')[0]
    },
    provider: 'test-provider', // Explicitly use test provider
    test_mode: true,           // Enable test mode
    no_send: true              // Ensure no real emails are sent
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': apiKey,
      'X-Test-Mode': 'true',       // Ensure test mode header
      'X-Test-Provider': 'true',   // Force test provider header 
      'X-No-Delivery': 'true'      // Prevent actual sending
    },
  };
  
  const res = http.post(`${config.urls.delivery}/api/render/email`, payload, params);
  helpers.handleResponse(res, 'delivery');
  
  return res;
}

// Test template management
export function testTemplateManagement(apiKey) {
  const templateId = `test-template-${generators.randomString(6)}`;
  
  // Create a test template
  const createPayload = JSON.stringify({
    template_id: templateId,
    name: `Test Template ${generators.randomString(4)}`,
    content: `<h1>Test Template</h1>
              <p>Hello {{user_name}},</p>
              <p>This is a test template created during load testing.</p>
              <p>Current date: {{current_date}}</p>
              <p>Click <a href="{{action_url}}">here</a> to take action.</p>
              <p>Best regards,<br>{{company_name}}</p>`,
    variables: ['user_name', 'company_name', 'action_url', 'current_date'],
    test_mode: true
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': apiKey,
      'X-Test-Mode': 'true'
    },
  };
  
  const createRes = http.post(`${config.urls.delivery}/api/templates`, createPayload, params);
  helpers.handleResponse(createRes, 'delivery');
  
  // If successful, try to get the template
  if (createRes.status === 201 || createRes.status === 200) {
    sleep(0.3);
    const getRes = http.get(`${config.urls.delivery}/api/templates/${templateId}`, params);
    helpers.handleResponse(getRes, 'delivery');
    return { templateId, res: getRes };
  }
  
  return { templateId, res: createRes };
}

// Test message simulation (simulating Kafka consumption)
export function testKafkaConsumption(apiKey) {
  // Simulate a message coming from Kafka
  const messageId = generators.uuid();
  const payload = JSON.stringify({
    message_id: messageId,
    priority: Math.floor(Math.random() * 5) + 1, // Priority 1-5
    recipient: generators.randomEmail(),
    template_id: 'default-template',
    data: {
      user_name: 'Test User',
      company_name: 'NOTLI Testing',
      action_url: 'https://example.com/test',
      current_date: new Date().toISOString().split('T')[0]
    },
    test_mode: true // Ensure no real email is sent
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': apiKey,
      'X-Test-Mode': 'true'
    },
  };
  
  const res = http.post(`${config.urls.delivery}/api/simulate/message`, payload, params);
  helpers.handleResponse(res, 'delivery');
  
  return res;
}

// Test delivery status
export function testDeliveryStatus(apiKey, messageId) {
  if (!messageId) {
    messageId = generators.uuid(); // Fallback to a random ID
  }
  
  const params = {
    headers: {
      'X-API-Key': apiKey
    },
  };
  
  const res = http.get(`${config.urls.delivery}/api/status/${messageId}`, params);
  helpers.handleResponse(res, 'delivery');
  
  return res;
}

// Main test function
export default function() {
  // Get API key from previous run or fallback to default
  const apiKey = helpers.getApiKey() || 'test-api-key-12345-abcdef';
  
  // Include test mode flag in the execution context
  __ENV.testMode = true;
  
  // 1. Always test health endpoint
  testHealth();
  sleep(0.3);
  
  // 2. Test email rendering
  const renderRes = testEmailRendering(apiKey);
  sleep(0.5);
  
  // 3. Test template management occasionally
  if (Math.random() < 0.15) { // 15% chance
    testTemplateManagement(apiKey);
    sleep(0.5);
  }
  
  // 4. Test Kafka message consumption
  let messageId = null;
  const consumeRes = testKafkaConsumption(apiKey);
  if (consumeRes.status === 200 || consumeRes.status === 202) {
    try {
      const body = JSON.parse(consumeRes.body);
      if (body && body.message_id) {
        messageId = body.message_id;
      }
    } catch (e) {
      console.log("Failed to parse message consumption response:", e);
    }
  }
  
  // 5. Test delivery status occasionally
  if (messageId && Math.random() < 0.3) { // 30% chance if we have a message ID
    sleep(0.4);
    testDeliveryStatus(apiKey, messageId);
  }
  
  // Random sleep between requests to simulate varying traffic patterns
  sleep(Math.random() * 1.5 + 0.5);
}
