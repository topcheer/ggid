import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TENANT_ID = __ENV.TENANT_ID || '00000000-0000-0000-0000-000000000001';

const errorRate = new Rate('errors');
const verifyDuration = new Trend('jwt_verify_duration');

export const options = {
  scenarios: {
    burst: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 100 },
        { duration: '30s', target: 100 },
        { duration: '10s', target: 0 },
      ],
    },
  },
  thresholds: {
    errors: ['rate<0.01'],
    jwt_verify_duration: ['p(95)<100', 'p(99)<200'],
    http_req_duration: ['p(95)<200'],
  },
};

export function setup() {
  const headers = {
    'Content-Type': 'application/json',
    'X-Tenant-ID': TENANT_ID,
  };

  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({
      username: 'admin',
      password: 'Admin@123456',
      tenant_id: TENANT_ID,
    }),
    { headers }
  );

  if (loginRes.status === 200) {
    const body = JSON.parse(loginRes.body);
    return { jwt: body.access_token || body.token || '' };
  }
  return { jwt: '' };
}

export default function (data) {
  const headers = {
    'X-Tenant-ID': TENANT_ID,
    Authorization: data.jwt ? `Bearer ${data.jwt}` : '',
  };

  group('JWT-protected endpoint', () => {
    const start = Date.now();
    const res = http.get(`${BASE_URL}/api/v1/users`, { headers });
    verifyDuration.add(Date.now() - start);

    errorRate.add(res.status === 401 || res.status === 500);
    check(res, {
      'authenticated': (r) => r.status === 200,
    });
  });
}
