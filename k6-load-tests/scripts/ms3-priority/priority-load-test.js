// Prioritization Engine (MS3) Load Test
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
    'priority_latency': ['p(95)<250'], // 95% of priority requests under 250ms
  },
};

// Test health endpoint
export function testHealth() {
  const res = http.get(`${config.urls.priority}/health`);
  helpers.handleResponse(res, 'priority');
  
  return res;
}

// Test priority rules
export function testPriorityRules(apiKey) {
  const payload = JSON.stringify({
    rule_name: `test-rule-${generators.randomString(8)}`,
    conditions: [
      {
        field: "recipient",
        operator: "contains",
        value: "vip"
      },
      {
        field: "subject",
        operator: "contains",
        value: "urgent"
      }
    ],
    priority_level: Math.floor(Math.random() * 5) + 1, // Priority 1-5
    description: "Test rule created during load testing"
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': apiKey
    },
  };
  
  const res = http.post(`${config.urls.priority}/api/rules`, payload, params);
  helpers.handleResponse(res, 'priority');
  
  // If successful, try to get the rule
  if (res.status === 201 || res.status === 200) {
    try {
      const body = JSON.parse(res.body);
      if (body && body.id) {
        // Get the created rule
        sleep(0.3);
        const getRes = http.get(`${config.urls.priority}/api/rules/${body.id}`, params);
        helpers.handleResponse(getRes, 'priority');
        return { ruleId: body.id, res: getRes };
      }
    } catch (e) {
      console.log("Failed to parse rule creation response:", e);
    }
  }
  
  return { ruleId: null, res };
}

// Test message simulation (to simulate Kafka messages coming in)
export function testMessageSimulation(apiKey) {
  // Create a simulated message payload
  const messageId = generators.uuid();
  const payload = JSON.stringify({
    message_id: messageId,
    channel: "email",
    recipient: Math.random() < 0.3 ? "vip@example.com" : generators.randomEmail(),
    subject: Math.random() < 0.2 ? "URGENT: Action Required" : `Test message ${generators.randomString(6)}`,
    body: `This is a simulated message for priority engine testing. ID: ${messageId}`,
    metadata: {
      source: "load_test",
      timestamp: new Date().toISOString()
    }
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': apiKey
    },
  };
  
  const res = http.post(`${config.urls.priority}/api/simulate/message`, payload, params);
  helpers.handleResponse(res, 'priority');
  
  return res;
}

// Test queue status
export function testQueueStatus(apiKey) {
  const params = {
    headers: {
      'X-API-Key': apiKey
    },
  };
  
  const res = http.get(`${config.urls.priority}/api/queue/status`, params);
  helpers.handleResponse(res, 'priority');
  
  return res;
}

// Main test function
export default function() {
  // Get API key from previous run or fallback to default
  const apiKey = helpers.getApiKey() || 'test-api-key-12345-abcdef';
  
  // 1. Always test health endpoint
  testHealth();
  sleep(0.3);
  
  // 2. Test priority rules occasionally
  if (Math.random() < 0.2) { // 20% chance
    const ruleResult = testPriorityRules(apiKey);
    sleep(0.5);
  }
  
  // 3. Test message simulation (always)
  testMessageSimulation(apiKey);
  sleep(0.4);
  
  // 4. Check queue status occasionally 
  if (Math.random() < 0.3) { // 30% chance
    testQueueStatus(apiKey);
  }
  
  // Random sleep between requests to simulate varying traffic patterns
  sleep(Math.random() * 1.5 + 0.5);
}
