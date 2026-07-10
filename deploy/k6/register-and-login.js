import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TENANT_ID = __ENV.TENANT_ID || '00000000-0000-0000-0000-000000000001';

const errorRate = new Rate('errors');
const loginDuration = new Trend('login_duration');
const registerDuration = new Trend('register_duration');

export const options = {
  stages: [
    { duration: '30s', target: 20 },   // ramp up to 20 VUs
    { duration: '1m', target: 50 },    // ramp up to 50 VUs
    { duration: '2m', target: 50 },    // hold at 50 VUs
    { duration: '30s', target: 0 },     // ramp down
  ],
  thresholds: {
    errors: ['rate<0.05'],               // < 5% errors
    http_req_duration: ['p(95)<500'],    // 95% of requests < 500ms
    login_duration: ['p(95)<300'],       // login 95th percentile < 300ms
    register_duration: ['p(95)<400'],    // register 95th percentile < 400ms
  },
};

export default function () {
  const headers = {
    'Content-Type': 'application/json',
    'X-Tenant-ID': TENANT_ID,
  };

  group('Register + Login flow', () => {
    const uniqueId = `${__VU}-${__ITER}-${Date.now()}`;

    // Register
    group('Register', () => {
      const start = Date.now();
      const res = http.post(
        `${BASE_URL}/api/v1/auth/register`,
        JSON.stringify({
          username: `benchuser_${uniqueId}`,
          email: `bench_${uniqueId}@example.com`,
          password: 'Bench@123456',
          full_name: 'Benchmark User',
          tenant_id: TENANT_ID,
        }),
        { headers }
      );
      registerDuration.add(Date.now() - start);
      errorRate.add(res.status >= 400);
      check(res, {
        'register success or conflict': (r) => r.status === 201 || r.status === 409,
      });
    });

    // Login
    let jwt = '';
    group('Login', () => {
      const start = Date.now();
      const res = http.post(
        `${BASE_URL}/api/v1/auth/login`,
        JSON.stringify({
          username: `benchuser_${uniqueId}`,
          password: 'Bench@123456',
          tenant_id: TENANT_ID,
        }),
        { headers }
      );
      loginDuration.add(Date.now() - start);
      errorRate.add(res.status >= 400);
      check(res, {
        'login success': (r) => r.status === 200,
      });

      if (res.status === 200) {
        const body = JSON.parse(res.body);
        jwt = body.access_token || body.token || '';
      }
    });

    // Authenticated request
    if (jwt) {
      group('List users (authenticated)', () => {
        const res = http.get(`${BASE_URL}/api/v1/users`, {
          headers: {
            ...headers,
            Authorization: `Bearer ${jwt}`,
          },
        });
        errorRate.add(res.status >= 400);
        check(res, {
          'list users success': (r) => r.status === 200,
        });
      });
    }
  });

  sleep(0.5);
}

export function handleSummary(data) {
  return {
    'stdout': JSON.stringify(data, null, 2),
    'deploy/k6/results.json': JSON.stringify(data, null, 2),
  };
}
