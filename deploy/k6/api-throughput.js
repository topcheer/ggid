import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Counter } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TENANT_ID = __ENV.TENANT_ID || '00000000-0000-0000-0000-000000000001';

const errorRate = new Rate('errors');
const requestsPerEndpoint = new Counter('requests_per_endpoint');

export const options = {
  scenarios: {
    sustained_load: {
      executor: 'constant-arrival-rate',
      rate: 100,          // 100 iterations/sec
      timeUnit: '1s',
      duration: '3m',
      preAllocatedVUs: 50,
      maxVUs: 200,
    },
  },
  thresholds: {
    errors: ['rate<0.05'],
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.05'],
  },
};

// Pre-login to get a JWT
let JWT = '';

export function setup() {
  const headers = {
    'Content-Type': 'application/json',
    'X-Tenant-ID': TENANT_ID,
  };

  // Try to register admin if not exists
  http.post(
    `${BASE_URL}/api/v1/auth/register`,
    JSON.stringify({
      username: 'admin',
      email: 'admin@example.com',
      password: 'Admin@123456',
      full_name: 'Admin',
      tenant_id: TENANT_ID,
    }),
    { headers }
  );

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
    'Content-Type': 'application/json',
    'X-Tenant-ID': TENANT_ID,
    Authorization: data.jwt ? `Bearer ${data.jwt}` : '',
  };

  const endpoints = [
    { method: 'GET', url: '/healthz', authenticated: false },
    { method: 'GET', url: '/api/v1/users', authenticated: true },
    { method: 'GET', url: '/api/v1/roles', authenticated: true },
    { method: 'GET', url: '/api/v1/orgs', authenticated: true },
    { method: 'GET', url: '/api/v1/audit', authenticated: true },
  ];

  const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];
  requestsPerEndpoint.add(1, { endpoint: endpoint.url });

  const res = http.request(endpoint.method, `${BASE_URL}${endpoint.url}`, null, {
    headers: endpoint.authenticated ? headers : { 'X-Tenant-ID': TENANT_ID },
  });

  errorRate.add(res.status >= 500);
  check(res, {
    'status not 5xx': (r) => r.status < 500,
  });

  sleep(0.1);
}
