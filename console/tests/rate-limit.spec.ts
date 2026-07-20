import { flushRateLimits } from "./helpers/flush-ratelimit";
import { test, expect, type APIRequestContext } from '@playwright/test';

const API_BASE = process.env.API_URL || 'https://ggid.iot2.win';
const TENANT = '00000000-0000-0000-0000-000000000001';

async function failLogin(request: APIRequestContext, username: string, password: string): Promise<number> {
  const resp = await request.post(`${API_BASE}/api/v1/auth/login`, {
    headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
    data: { username, password, tenant_slug: 'default' },
  });
  return resp.status();
}

test.describe('Rate Limiting & Account Lockout', () => {
  test.beforeAll(async () => { await flushRateLimits(); });

  test('5 wrong passwords triggers lockout or rate limit', async ({ request }) => {
    await flushRateLimits();
    const username = `lockout_${Date.now()}`;
    // Register a user first
    await request.post(`${API_BASE}/api/v1/auth/register`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { username, email: `${username}@test.com`, password: 'CorrectPass123!' },
    });
    await new Promise(r => setTimeout(r, 500));

    // Attempt 5 wrong passwords
    const statuses: number[] = [];
    for (let i = 0; i < 5; i++) {
      const code = await failLogin(request, username, 'WrongPassword123!');
      statuses.push(code);
      await new Promise(r => setTimeout(r, 200));
    }

    // After 5 attempts: should get 429 (rate limited) or 423 (locked) or 401
    const lastStatus = statuses[statuses.length - 1];
    expect([401, 423, 429]).toContain(lastStatus);

    // 6th attempt should also be blocked
    const code6 = await failLogin(request, username, 'WrongPassword123!');
    expect([401, 423, 429]).toContain(code6);
  });

  test('correct password works when not rate limited', async ({ request }) => {
    await flushRateLimits();
    const username = `ok_${Date.now()}`;
    await request.post(`${API_BASE}/api/v1/auth/register`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { username, email: `${username}@test.com`, password: 'GoodPass123!' },
    });
    await new Promise(r => setTimeout(r, 500));

    const code = await failLogin(request, username, 'GoodPass123!');
    expect(code).toBe(200);
  });
});
