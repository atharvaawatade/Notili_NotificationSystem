// NOTLI K6 Load Test Suite - Main Entry Point
import { sleep } from 'k6';
import http from 'k6/http';
import { config } from './config.js';
import { metrics, helpers } from './utils.js';

// Import individual test scenarios
import { default as authTest } from './ms1-auth/auth-load-test.js';
import { default as queueTest } from './ms2-queue/queue-load-test.js';
import { default as priorityTest } from './ms3-priority/priority-load-test.js';
import { default as deliveryTest } from './ms4-delivery/delivery-load-test.js';

// Default configuration - can be overridden via environment variables
export const options = {
  // Default to the load profile (can be changed via environment variables)
  stages: __ENV.PROFILE ? config.profiles[__ENV.PROFILE].stages : config.profiles.load.stages,
  thresholds: config.thresholds,
  // Add tags for better organization in Grafana Cloud or other dashboards
  tags: {
    service: __ENV.SERVICE || 'all',
    environment: __ENV.ENV || 'dev'
  }
};

// Main test function that orchestrates the overall test flow
export default function() {
  // Get or determine which service to test
  const service = __ENV.SERVICE || 'all';
  
  // Test health endpoints for all services first
  testAllHealth();
  
  // Run tests based on the selected service
  switch(service.toLowerCase()) {
    case 'ms1':
    case 'auth':
      authTest();
      break;
    case 'ms2':
    case 'queue':
      queueTest();
      break;
    case 'ms3':
    case 'priority':
      priorityTest();
      break;
    case 'ms4':
    case 'delivery':
      deliveryTest();
      break;
    case 'all':
    default:
      // Test all services with a weighted distribution
      // Focus more on the message queue (MS2) with our weighted approach
      const rand = Math.random();
      if (rand < 0.35) {
        // 35% of traffic goes to the queue service (MS2) - our primary focus
        queueTest();
      } else if (rand < 0.6) {
        // 25% goes to auth service (MS1)
        authTest();
      } else if (rand < 0.8) {
        // 20% goes to priority service (MS3)
        priorityTest();
      } else {
        // 20% goes to delivery service (MS4)
        deliveryTest();
      }
      break;
  }
}

// Test health endpoints for all services
function testAllHealth() {
  // Only run health checks occasionally to reduce noise
  if (Math.random() > 0.3) return;
  
  // Check health endpoints for all services
  const responses = http.batch([
    ['GET', `${config.urls.auth}/health`],
    ['GET', `${config.urls.queue}/health`],
    ['GET', `${config.urls.priority}/health`],
    ['GET', `${config.urls.delivery}/health`]
  ]);
  
  // Process responses
  responses.forEach((res, index) => {
    const serviceNames = ['auth', 'queue', 'priority', 'delivery'];
    helpers.handleResponse(res, serviceNames[index]);
  });
  
  // Small delay after health checks
  sleep(0.2);
}
