// Authentication Service (MS1) Load Test
import { sleep, check } from 'k6';
import http from 'k6/http';
import { config } from '../config.js';
import { metrics, generators, helpers } from '../utils.js';

// Default test configuration - can be overridden via environment variables
export const options = {
  // Use the stress test profile by default
  stages: config.profiles.stress.stages,
  thresholds: config.thresholds,
};

// Global variables to store test state
let apiKey = '';

// Signup test
export function testSignup() {
  const email = generators.randomEmail();
  const password = generators.randomPassword();
  
  const payload = JSON.stringify({
    email: email,
    password: password,
    name: 'Test User'
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };
  
  const res = http.post(`${config.urls.auth}/api/auth/signup`, payload, params);
  helpers.handleResponse(res, 'auth');
  
  return { email, password, res };
}

// Login test
export function testLogin(email, password) {
  const payload = JSON.stringify({
    email: email,
    password: password
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };
  
  const res = http.post(`${config.urls.auth}/api/auth/login`, payload, params);
  helpers.handleResponse(res, 'auth');
  
  if (res.status === 200) {
    // Parse and store auth token if needed
    try {
      const body = JSON.parse(res.body);
      if (body.token) {
        return body.token;
      }
    } catch (e) {
      console.log("Failed to parse login response:", e);
    }
  }
  
  return null;
}

// Generate API key
export function testGenerateApiKey(token) {
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
  };
  
  const res = http.post(`${config.urls.auth}/api/keys/generate`, null, params);
  helpers.handleResponse(res, 'auth');
  
  if (res.status === 200) {
    try {
      const body = JSON.parse(res.body);
      if (body.api_key) {
        apiKey = body.api_key;
        helpers.storeApiKey(apiKey);
        return apiKey;
      }
    } catch (e) {
      console.log("Failed to parse API key response:", e);
    }
  }
  
  return null;
}

// Validate API key
export function testValidateApiKey() {
  const key = apiKey || helpers.getApiKey() || '';
  
  const params = {
    headers: {
      'X-API-Key': key
    },
  };
  
  const res = http.get(`${config.urls.auth}/api/keys/validate`, params);
  helpers.handleResponse(res, 'auth');
  
  return res.status === 200;
}

// Main test function
export default function() {
  // Get API key from previous run or generate a new one
  apiKey = helpers.getApiKey();
  
  if (!apiKey) {
    // Need to create a new user and generate API key
    const signupResult = testSignup();
    
    // Wait a bit between signup and login
    sleep(1);
    
    if (signupResult.res.status === 201 || signupResult.res.status === 200) {
      const token = testLogin(signupResult.email, signupResult.password);
      
      if (token) {
        sleep(0.5);
        apiKey = testGenerateApiKey(token);
      }
    }
  } else {
    // Just validate existing API key
    testValidateApiKey();
  }
  
  // Random sleep between requests to simulate user behavior
  sleep(Math.random() * 2 + 1);
}
