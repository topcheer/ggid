import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 },
    { duration: '1m', target: 100 },
    { duration: '30s', target: 200 },
    { duration: '1m', target: 200 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<100'],
    http_req_failed: ['rate<0.02'],
  },
};

const BASE = __ENV.BASE_URL || 'http://localhost:8080';
const TOKEN = __ENV.AUTH_TOKEN || 'test-token';

export default function () {
  const res = http.post(`${BASE}/api/v1/policy/risk/evaluate`, JSON.stringify({
    user_id: `user-${__VU}`,
    session_id: `sess-${__VU}-${__ITER}`,
    context: {
      ip: `10.0.${__VU % 255}.${__ITER % 255}`,
      device_id: `dev-${__VU % 50}`,
    },
  }), {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${TOKEN}`,
    },
  });

  check(res, {
    'status 200': (r) => r.status === 200,
    'has score': (r) => r.json('score') !== undefined,
  });
  sleep(0.1);
}
