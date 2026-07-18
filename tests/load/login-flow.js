import http from 'k6/http';
import { check, sleep } from 'k6';

// Multi-stage load test for GGID login flow
// Usage: k6 run tests/load/login-flow.js
// Custom: k6 run -e BASE_URL=https://ggid.iot2.win tests/load/login-flow.js

const VUS = parseInt(__ENV.VUS) || 50;
const DURATION = __ENV.DURATION || '1m';

export const options = {
  scenarios: {
    steady_50: {
      executor: 'constant-vus',
      vus: 50,
      duration: '1m',
      tags: { scenario: '50vu' },
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<200', 'p(99)<500'],
    http_req_failed: ['rate<0.05'],
  },
};

const BASE = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const res = http.post(`${BASE}/api/v1/auth/login`, JSON.stringify({
    email: 'admin@ggid.local',
    password: 'Admin@123456',
  }), { headers: { 'Content-Type': 'application/json' } });

  check(res, {
    'status 200': (r) => r.status === 200,
    'has token': (r) => r.json('access_token') !== undefined,
    'latency < 200ms': (r) => r.timings.duration < 200,
  });
  sleep(0.1);
}
