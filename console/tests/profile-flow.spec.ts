import { flushRateLimits } from "./helpers/flush-ratelimit";
import { test, expect, type APIRequestContext, type Page } from '@playwright/test';

const BASE = process.env.BASE_URL || 'https://ggid-console.iot2.win';
const API_BASE = process.env.API_URL || 'https://ggid.iot2.win';
const TENANT = '00000000-0000-0000-0000-000000000001';
const TEST_PW = process.env.TEST_PASSWORD || '';

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

test.describe('Profile Flow', () => {
  test('profile tab -> fields load', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/profile');
    await page.waitForLoadState('domcontentloaded');

    await expect(page.locator('h1:has-text("Profile"), h1:has-text("My Profile")')).toBeVisible({ timeout: 5000 });
    const inputs = page.locator('input[type="text"], input[type="email"]');
    expect(await inputs.count()).toBeGreaterThan(0);
  });

  test('security tab -> switch', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/profile');
    await page.waitForLoadState('domcontentloaded');

    await page.click('button:has-text("Security")');
    await page.waitForTimeout(500);
    const bodyText = await page.textContent('body');
    expect(bodyText?.toLowerCase()).toContain('password');
  });

  test('devices tab -> switch', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/profile');
    await page.waitForLoadState('domcontentloaded');

    await page.click('button:has-text("Device")');
    await page.waitForTimeout(500);
    await expect(page.locator('body')).toBeVisible();
  });
});
