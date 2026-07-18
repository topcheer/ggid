import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 50 },
    { duration: '2m', target: 200 },
    { duration: '30s', target: 500 },
    { duration: '1m', target: 500 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<50'],
    http_req_failed: ['rate<0.01'],
  },
};

const BASE = __ENV.BASE_URL || 'http://localhost:8080';
const TOKEN = __ENV.AUTH_TOKEN || 'test-token';

export default function () {
  const res = http.post(`${BASE}/api/v1/policy/authorize`, JSON.stringify({
    subject: `user:${__VU}`,
    resource: `doc:${__ITER % 100}`,
    action: 'read',
    context: { tenant_id: 'default' },
  }), {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${TOKEN}`,
    },
  });

  check(res, {
    'status 200': (r) => r.status === 200,
    'has decision': (r) => r.json('decision') !== undefined,
  });
  sleep(0.05);
}
