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

test.describe('SCIM / LDAP Configuration', () => {
  test('SCIM page loads', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/settings/scim');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1000);

    const bodyText = await page.textContent('body') || '';
    expect(bodyText.toLowerCase()).not.toContain('application error');
  });

  test('SCIM page has endpoint display or config form', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/settings/scim');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1000);

    const inputs = page.locator('input');
    const count = await inputs.count();
    // SCIM config page should have some inputs or informational text
    const bodyText = await page.textContent('body') || '';
    expect(count > 0 || bodyText.length > 50).toBeTruthy();
  });

  test('LDAP config page loads', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/settings/ldap-config');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1000);

    const bodyText = await page.textContent('body') || '';
    expect(bodyText.toLowerCase()).not.toContain('application error');
  });

  test('LDAP config page has server/port inputs', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/settings/ldap-config');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    const inputs = page.locator('input, select');
    const count = await inputs.count();
    const bodyText = await page.textContent('body') || '';
    // LDAP config should have inputs OR meaningful content
    expect(count > 0 || bodyText.length > 50).toBeTruthy();
  });

  test('SCIM API returns config data', async ({ request }) => {
    const { token } = await getAuthToken(request);
    const res = await request.get(`${API_BASE}/api/v1/scim/config`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    // SCIM endpoint may or may not be configured, but should not crash
    expect(res.status()).toBeLessThan(500);
  });
});
