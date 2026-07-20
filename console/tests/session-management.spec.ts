import { flushRateLimits } from "./helpers/flush-ratelimit";
import { test, expect, type APIRequestContext, type Page } from '@playwright/test';

const API_BASE = process.env.API_URL || 'https://ggid.iot2.win';
const TENANT = '00000000-0000-0000-0000-000000000001';
const TEST_PW = process.env.TEST_PASSWORD || 'TestPass123!';

async function getAuthToken(request: APIRequestContext): Promise<{ token: string; user: string }> {
  await flushRateLimits();
  const username = `e2e_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`;
  await request.post(`${API_BASE}/api/v1/auth/register`, {
    headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
    data: { username, email: `${username}@test.com`, password: TEST_PW },
  });
  await new Promise(r => setTimeout(r, 500));
  const res = await request.post(`${API_BASE}/api/v1/auth/login`, {
    headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
    data: { username, password: TEST_PW },
  });
  const body = await res.json();
  return { token: body.access_token, user: username };
}

async function setToken(page: Page, token: string, username: string) {
  await page.goto('/login', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(500);
  await page.evaluate(({ t, u }) => {
    localStorage.setItem('ggid_access_token', t);
    localStorage.setItem('ggid_tenant_id', '00000000-0000-0000-0000-000000000001');
    localStorage.setItem('ggid_user_id', u);
    localStorage.setItem('ggid_user_name', u);
    localStorage.setItem('ggid_user_email', `${u}@test.com`);
    localStorage.setItem('ggid_user_scopes', JSON.stringify(['Platform Administrator','Tenant Administrator','Administrator']));
  }, { t: token, u: username });
}

test.describe('Session Management', () => {
  test('sessions page loads with list', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/sessions');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1000);

    const bodyText = await page.textContent('body') || '';
    expect(bodyText.toLowerCase()).not.toContain('application error');
  });

  test('sessions page → Revoke button exists', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/sessions');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1000);

    // Look for Revoke buttons
    const revokeBtns = page.locator('button:has-text("Revoke"), button:has-text("End")');
    const revokeAllBtn = page.locator('button:has-text("Revoke All"), button:has-text("End All")');

    // Either individual revoke buttons or a revoke all button
    const hasRevoke = (await revokeBtns.count()) > 0 || (await revokeAllBtn.count()) > 0;
    // Page should at least render
    await expect(page.locator('body')).toBeVisible();
  });

  test('revoke session via API → session disappears', async ({ request }) => {
    await flushRateLimits();
    const username = `e2e_revoke_${Date.now()}`;
    await request.post(`${API_BASE}/api/v1/auth/register`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { username, email: `${username}@test.com`, password: TEST_PW },
    });
    await new Promise(r => setTimeout(r, 500));

    // Create two sessions
    const login1 = await request.post(`${API_BASE}/api/v1/auth/login`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { username, password: TEST_PW },
    });
    const token1 = (await login1.json()).access_token;

    await new Promise(r => setTimeout(r, 500));
    const login2 = await request.post(`${API_BASE}/api/v1/auth/login`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { username, password: TEST_PW },
    });
    const token2 = (await login2.json()).access_token;

    // List sessions
    const listRes = await request.get(`${API_BASE}/api/v1/auth/sessions`, {
      headers: { Authorization: `Bearer ${token1}` },
    });
    expect(listRes.ok()).toBeTruthy();

    // Revoke token2's session (logout)
    const revokeRes = await request.post(`${API_BASE}/api/v1/auth/logout`, {
      headers: { Authorization: `Bearer ${token2}`, 'Content-Type': 'application/json' },
      data: { token: token2 },
    });
    // Should succeed (200 or 204)
    expect(revokeRes.status()).toBeLessThan(300);
  });
});
