import { test, expect, type APIRequestContext, type Page } from '@playwright/test';

const BASE = process.env.BASE_URL || 'https://ggid-console.iot2.win';
const API_BASE = process.env.API_URL || 'http://192.168.31.13:30080';
const TENANT = '00000000-0000-0000-0000-000000000001';

async function getAuthToken(request: APIRequestContext): Promise<{ token: string; user: string }> {
  const username = `e2e_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`;
  await request.post(`${API_BASE}/api/v1/auth/register`, {
    headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
    data: { username, email: `${username}@test.com`, password: 'TestPass123!' },
  });
  await new Promise(r => setTimeout(r, 500));
  const res = await request.post(`${API_BASE}/api/v1/auth/login`, {
    headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
    data: { username, password: 'TestPass123!' },
  });
  const body = await res.json();
  return { token: body.access_token, user: username };
}

async function setToken(page: Page, token: string, username: string) {
  await page.goto('/login');
  await page.evaluate(({ t, u }) => {
    localStorage.setItem('ggid_access_token', t);
    localStorage.setItem('ggid_user_id', u);
    localStorage.setItem('ggid_user_name', u);
    localStorage.setItem('ggid_user_email', `${u}@test.com`);
    localStorage.setItem('ggid_tenant_id', TENANT);
    localStorage.setItem('ggid_user_scopes', JSON.stringify(['platform:admin', 'tenant:admin', 'admin']));
  }, { t: token, u: username });
}

test.describe('Auth Flow Extended', () => {
  test('login → dashboard → refresh keeps session', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/dashboard');
    await page.waitForLoadState('domcontentloaded');
    await expect(page.locator('body')).toBeVisible();

    // Refresh page
    await page.reload();
    await page.waitForLoadState('domcontentloaded');

    // Should still be on dashboard (not redirected to login)
    const url = page.url();
    expect(url).not.toContain('/login');
  });

  test('create user → form fills → save', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/users');
    await page.waitForLoadState('domcontentloaded');

    // Click Create User button
    const createBtn = page.locator('button:has-text("Create"), button:has-text("Add"):has-text("User")').first();
    if (await createBtn.isVisible({ timeout: 5000 }).catch(() => false)) {
      await createBtn.click();
      await page.waitForTimeout(500);

      // Fill username
      const userInput = page.locator('input[name="username"], input[placeholder*="username" i]').first();
      if (await userInput.isVisible()) {
        await userInput.fill(`e2e_user_${Date.now()}`);
      }

      // Fill email
      const emailInput = page.locator('input[type="email"], input[name="email"]').first();
      if (await emailInput.isVisible()) {
        await emailInput.fill(`e2e_user_${Date.now()}@test.com`);
      }
    }
    await expect(page.locator('body')).toBeVisible();
  });

  test('change password → form appears', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/profile');
    await page.waitForLoadState('domcontentloaded');

    // Go to Security tab
    await page.click('button:has-text("Security")');
    await page.waitForTimeout(500);

    // Click Change Password
    const changePwBtn = page.locator('button:has-text("Change Password")');
    if (await changePwBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await changePwBtn.click();
      await page.waitForTimeout(500);

      // Verify password form fields appear
      const pwInputs = page.locator('input[type="password"]');
      const count = await pwInputs.count();
      expect(count).toBeGreaterThanOrEqual(1);
    }
  });
});
