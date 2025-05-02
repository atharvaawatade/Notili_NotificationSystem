// NOTLI Microservices Load Testing Utilities
import { check } from 'k6';
import http from 'k6/http';
import { Trend, Rate, Counter } from 'k6/metrics';

// Custom metrics for detailed analysis
export const metrics = {
  successRate: new Rate('success_rate'),
  errorRate: new Rate('error_rate'),
  authLatency: new Trend('auth_latency', true),
  queueLatency: new Trend('queue_latency', true),
  priorityLatency: new Trend('priority_latency', true),
  deliveryLatency: new Trend('delivery_latency', true),
  requestsCounter: new Counter('total_requests'),
};

// Helper for generating random test data
export const generators = {
  // Generate random email address
  randomEmail: () => {
    const random = Math.random().toString(36).substring(2, 10);
    return `test-${random}@notli.example.com`;
  },
  
  // Generate random password (8-12 chars)
  randomPassword: () => {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*';
    const length = Math.floor(Math.random() * 5) + 8; // 8-12 characters
    let password = '';
    for (let i = 0; i < length; i++) {
      password += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return password;
  },
  
  // Generate random string of specified length
  randomString: (length = 10) => {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let result = '';
    for (let i = 0; i < length; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result;
  },
  
  // Generate UUID for idempotency keys
  uuid: () => {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
      const r = Math.random() * 16 | 0, v = c === 'x' ? r : (r & 0x3 | 0x8);
      return v.toString(16);
    });
  }
};

// Helper functions for common operations
export const helpers = {
  // Handle HTTP response and record metrics
  handleResponse: (res, serviceName) => {
    // Check if response was successful
    const success = check(res, {
      'status is 2xx': (r) => r.status >= 200 && r.status < 300,
    });
    
    // Record custom metrics based on service name
    metrics.successRate.add(success);
    metrics.errorRate.add(!success);
    metrics.requestsCounter.add(1);
    
    // Add latency to appropriate service metric
    switch(serviceName) {
      case 'auth':
        metrics.authLatency.add(res.timings.duration);
        break;
      case 'queue':
        metrics.queueLatency.add(res.timings.duration);
        break;
      case 'priority':
        metrics.priorityLatency.add(res.timings.duration);
        break;
      case 'delivery':
        metrics.deliveryLatency.add(res.timings.duration);
        break;
    }
    
    // Log errors for troubleshooting
    if (!success) {
      console.log(`Error in ${serviceName} service: Status ${res.status}, Body: ${res.body}`);
    }
    
    return success;
  },
  
  // Store API key between test iterations
  storeApiKey: (apiKey) => {
    // Make API key available globally
    if (!__ENV.apiKey) {
      __ENV.apiKey = apiKey;
    }
    return apiKey;
  },
  
  // Get stored API key
  getApiKey: () => {
    return __ENV.apiKey || null;
  }
};
