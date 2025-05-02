// MS2 Message Queue Service - Breaking Point Detection Test
// This script is focused on finding the exact breaking point of MS2
import { sleep, check } from 'k6';
import http from 'k6/http';
import { Counter, Rate, Trend } from 'k6/metrics';
import { config } from '../config.js';
import { metrics, generators, helpers } from '../utils.js';

// Custom metrics specific to breaking point detection
const errorsByCode = {
  '429': new Counter('errors_rate_limited'),  // Rate limiting errors
  '500': new Counter('errors_server'),        // Server errors
  '502': new Counter('errors_bad_gateway'),   // Bad gateway
  '503': new Counter('errors_service_unavailable'), // Service unavailable
  '504': new Counter('errors_gateway_timeout'),     // Gateway timeout
};

// Response time metrics bucketed by VU count
const responseTimeByLoad = {
  bucket100: new Trend('response_time_100_vu'),
  bucket500: new Trend('response_time_500_vu'),
  bucket1000: new Trend('response_time_1000_vu'),
  bucket1500: new Trend('response_time_1500_vu'),
  bucket2000: new Trend('response_time_2000_vu'),
  bucket2500: new Trend('response_time_2500_vu'),
  bucket3000: new Trend('response_time_3000_vu'),
};

// Advanced test configuration for breaking point detection
export const options = {
  // Use the specialized MS2 breakpoint profile
  stages: config.profiles.ms2_breakpoint.stages,
  
  // More detailed thresholds for breaking point analysis
  thresholds: {
    'http_req_failed': ['rate<0.5'], // Allow up to 50% failure rate as we're finding the breaking point
    'http_req_duration': ['p(95)<1000'], // 95% of requests under 1000ms (1 second)
    'error_rate': ['value<0.5'],     // Error rate below 50%
  },
  
  // System resource monitoring
  systemTags: ['cpu', 'mem'],
  
  // Additional configuration
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)', 'p(99.9)'],
  noConnectionReuse: false,
  userAgent: 'K6BreakpointTest/1.0',
};

// Track the active VUs and total errors to determine breaking point
let currentVUs = 0;
let totalRequests = 0;
let totalErrors = 0;
let breakingPointDetected = false;
let breakingPointVUs = 0;
let breakingPointErrorRate = 0;
let breakingPointTime = null;

// Main test function for MS2 breaking point detection
export default function() {
  // Update current VU count
  currentVUs = __VU;
  
  // Get API key from environment variable (passed in via command line)
  const apiKey = __ENV.API_KEY || 'TEST_KEY_PLACEHOLDER';
  if (apiKey === 'TEST_KEY_PLACEHOLDER') {
    console.error('⚠️ WARNING: No API key provided. Use -e API_KEY=your_key when running the test');
  }
  
  // Create a unique idempotency key for each request
  const idempotencyKey = generators.uuid();
  
  // Create realistic message payload
  const payload = JSON.stringify({
    idempotency_key: idempotencyKey,
    recipient: generators.randomEmail(),
    subject: `MS2 Breaking Point Test ${generators.randomString(8)}`,
    body: `This is a test email for breaking point detection. Current VUs: ${currentVUs}, Request ID: ${idempotencyKey}`,
    from_name: 'NOTLI Load Test',
    reply_to: ['noreply@notli.example.com'],
    channel_data: {
      attachments: [],
      cc: [],
      bcc: []
    }
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': apiKey,
      'X-Test-Request-ID': idempotencyKey,
      'X-Request-VUs': currentVUs.toString()
    },
    tags: {
      vus: currentVUs
    }
  };
  
  // Start request timing
  const startTime = new Date().getTime();
  
  // Make the request to MS2's notify endpoint
  const res = http.post(`${config.urls.queue}/v1/email/notify`, payload, params);
  
  // End request timing
  const endTime = new Date().getTime();
  const duration = endTime - startTime;
  
  // Track statistics
  totalRequests++;
  
  // Track response time in the appropriate VU bucket
  if (currentVUs <= 100) {
    responseTimeByLoad.bucket100.add(duration);
  } else if (currentVUs <= 500) {
    responseTimeByLoad.bucket500.add(duration);
  } else if (currentVUs <= 1000) {
    responseTimeByLoad.bucket1000.add(duration);
  } else if (currentVUs <= 1500) {
    responseTimeByLoad.bucket1500.add(duration);
  } else if (currentVUs <= 2000) {
    responseTimeByLoad.bucket2000.add(duration);
  } else if (currentVUs <= 2500) {
    responseTimeByLoad.bucket2500.add(duration);
  } else {
    responseTimeByLoad.bucket3000.add(duration);
  }
  
  // Check success and log any errors
  const success = res.status >= 200 && res.status < 300;
  
  // Track error by status code if not successful
  if (!success) {
    totalErrors++;
    
    // Track specific error types
    if (errorsByCode[res.status]) {
      errorsByCode[res.status].add(1);
    }
    
    // Log errors if they happen at a reasonable rate (don't flood logs)
    if (Math.random() < 0.05) { // Log ~5% of errors
      console.log(
        `Error: ${res.status}, VUs: ${currentVUs}, ` +
        `Duration: ${duration}ms, Error rate: ${(totalErrors / totalRequests * 100).toFixed(2)}%`
      );
    }
    
    // Detect breaking point based on error rate
    const currentErrorRate = totalErrors / totalRequests;
    if (!breakingPointDetected && currentErrorRate > 0.1 && currentVUs > 500) {
      // If error rate exceeds 10% and we're past 500 VUs, consider this a breaking point
      breakingPointDetected = true;
      breakingPointVUs = currentVUs;
      breakingPointErrorRate = currentErrorRate;
      breakingPointTime = new Date().toISOString();
      
      console.log(`
      ===================== BREAKING POINT DETECTED =====================
      🚨 MS2 Breaking Point: ${breakingPointVUs} Virtual Users
      📉 Error Rate: ${(breakingPointErrorRate * 100).toFixed(2)}%
      ⏱️ Response Time: ${duration}ms
      🕒 Time: ${breakingPointTime}
      ================================================================
      `);
    }
  }
  
  // Add to standard metrics
  helpers.handleResponse(res, 'queue');
  
  // Dynamic sleep based on response - slow down more if we're getting errors
  if (!success) {
    sleep(Math.random() * 2 + 1); // 1-3 seconds on error
  } else {
    sleep(Math.random() * 0.5 + 0.2); // 0.2-0.7 seconds on success
  }
}

// Output the final breaking point results at the end of the test
export function handleSummary(data) {
  const result = {
    breaking_point: breakingPointDetected ? breakingPointVUs : "Not detected",
    error_rate_at_breaking_point: breakingPointDetected ? 
      (breakingPointErrorRate * 100).toFixed(2) + "%" : "N/A",
    time_at_breaking_point: breakingPointDetected ? breakingPointTime : "N/A",
    total_requests: totalRequests,
    total_errors: totalErrors,
    overall_error_rate: (totalErrors / totalRequests * 100).toFixed(2) + "%",
    http_req_duration: {
      min: data.metrics.http_req_duration.values.min,
      avg: data.metrics.http_req_duration.values.avg,
      median: data.metrics.http_req_duration.values.med,
      p90: data.metrics.http_req_duration.values["p(90)"],
      p95: data.metrics.http_req_duration.values["p(95)"],
      p99: data.metrics.http_req_duration.values["p(99)"],
      max: data.metrics.http_req_duration.values.max
    }
  };
  
  console.log("=============== MS2 BREAKING POINT TEST RESULTS ===============");
  console.log(JSON.stringify(result, null, 2));
  console.log("================================================================");
  
  return {
    "stdout": JSON.stringify(data),
    "./results/ms2-breakpoint-summary.json": JSON.stringify(result, null, 2),
    "./results/ms2-breakpoint-full.json": JSON.stringify(data, null, 2)
  };
}
