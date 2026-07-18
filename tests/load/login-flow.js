import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 },
    { duration: '1m', target: 50 },
    { duration: '30s', target: 100 },
    { duration: '1m', target: 100 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<200'],
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
  });
  sleep(0.1);
}
