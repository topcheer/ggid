import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 10 },
    { duration: '2m', target: 50 },
    { duration: '30s', target: 100 },
    { duration: '1m', target: 100 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<150'],
    http_req_failed: ['rate<0.05'],
  },
};

const BASE = __ENV.BASE_URL || 'http://localhost:8080';
const TOKEN = __ENV.AUTH_TOKEN || 'test-token';
const headers = {
  'Content-Type': 'application/json',
  'Authorization': `Bearer ${TOKEN}`,
};

export default function () {
  // Create user
  const createRes = http.post(`${BASE}/api/v1/identity/users`, JSON.stringify({
    username: `loadtest-${__VU}-${__ITER}`,
    email: `lt-${__VU}-${__ITER}@test.local`,
    password: 'TestPass123!',
  }), { headers });

  check(createRes, { 'create 200/201': (r) => r.status === 200 || r.status === 201 });

  // List users
  const listRes = http.get(`${BASE}/api/v1/identity/users?limit=20`, { headers });
  check(listRes, { 'list 200': (r) => r.status === 200 });

  sleep(0.2);
}
