// NOTLI Microservices Load Testing Configuration
export const config = {
  // Server endpoints
  urls: {
    auth: 'http://localhost:8081',        // Authentication Service (MS1)
    queue: 'http://localhost:8082',       // Message Queue Service (MS2)
    priority: 'http://localhost:8083',    // Prioritization Engine (MS3)
    delivery: 'http://localhost:8084',    // Delivery Engine (MS4)
  },
  
  // Load testing profiles
  profiles: {
    smoke: {
      vus: 1,
      duration: '10s',
      description: 'Quick verification test with minimal load'
    },
    load: {
      stages: [
        { duration: '30s', target: 100 },  // Ramp up to 100 VUs
        { duration: '1m', target: 100 },   // Stay at 100 VUs
        { duration: '30s', target: 0 },    // Ramp down
      ],
      description: 'Normal load simulation'
    },
    stress: {
      stages: [
        { duration: '30s', target: 200 },   // Ramp up to 200 VUs
        { duration: '1m', target: 500 },    // Ramp up to 500 VUs
        { duration: '1m', target: 1000 },   // Ramp up to 1000 VUs 
        { duration: '2m', target: 1500 },   // Push to 1500 VUs
        { duration: '2m', target: 2000 },   // Push to 2000 VUs
        { duration: '2m', target: 0 },      // Ramp down
      ],
      description: 'High load simulation to find breaking points'
    },
    ms2_breakpoint: {
      stages: [
        { duration: '20s', target: 500 },   // Ramp up to 500 VUs
        { duration: '40s', target: 1000 },  // Ramp up to 1000 VUs
        { duration: '60s', target: 1500 },  // Ramp up to 1500 VUs
        { duration: '60s', target: 2000 },  // Push to 2000 VUs
        { duration: '60s', target: 2500 },  // Push to 2500 VUs
        { duration: '60s', target: 3000 },  // Push to 3000 VUs
        { duration: '20s', target: 0 },     // Ramp down
      ],
      description: 'MS2 breaking point determination test',
      summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
    },
    spike: {
      stages: [
        { duration: '10s', target: 100 },   // Ramp up to 100 VUs
        { duration: '1m', target: 100 },    // Stay at 100 VUs
        { duration: '10s', target: 1500 },  // Spike to 1500 VUs
        { duration: '1m', target: 1500 },   // Stay at 1500 VUs
        { duration: '10s', target: 100 },   // Back to 100 VUs
        { duration: '1m', target: 100 },    // Stay at 100 VUs
        { duration: '10s', target: 0 },     // Ramp down
      ],
      description: 'Spike test to simulate sudden traffic surges'
    },
    soak: {
      stages: [
        { duration: '2m', target: 400 },    // Ramp up to 400 VUs
        { duration: '3h', target: 400 },    // Stay at 400 VUs for 3 hours
        { duration: '2m', target: 0 },      // Ramp down
      ],
      description: 'Extended load test to find memory leaks and stability issues'
    }
  },
  
  // Thresholds for pass/fail criteria
  thresholds: {
    http_req_failed: ['rate<0.01'],     // Less than 1% failure rate
    http_req_duration: ['p(95)<500'],   // 95% of requests under 500ms
  },
  
  // Test data
  testData: {
    email: 'test@example.com',
    password: 'Test@123',
    apiKeyPrefix: 'test-api-key-',
  }
};
