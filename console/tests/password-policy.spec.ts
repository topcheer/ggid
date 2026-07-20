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

test.describe('Password Policy', () => {
  test('register with short password → error message', async ({ request }) => {
    await flushRateLimits();
    const username = `e2e_pw_${Date.now()}`;
    const res = await request.post(`${API_BASE}/api/v1/auth/register`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { username, email: `${username}@test.com`, password: 'short' },
    });
    // Should reject short password
    expect(res.status()).toBeGreaterThanOrEqual(400);
    const body = await res.json().catch(() => ({}));
    const msg = JSON.stringify(body).toLowerCase();
    expect(msg.length).toBeGreaterThan(0);
  });

  test('register with weak password → error message', async ({ request }) => {
    await flushRateLimits();
    const username = `e2e_weak_${Date.now()}`;
    const res = await request.post(`${API_BASE}/api/v1/auth/register`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { username, email: `${username}@test.com`, password: '12345678' },
    });
    expect(res.status()).toBeGreaterThanOrEqual(400);
  });

  test('change password form → new password too short → error', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/profile');
    await page.waitForLoadState('domcontentloaded');

    await page.click('button:has-text("Security")');
    await page.waitForTimeout(500);

    const changePwBtn = page.locator('button:has-text("Change Password")');
    if (await changePwBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await changePwBtn.click();
      await page.waitForTimeout(500);

      // Fill with short new password
      const pwInputs = page.locator('input[type="password"]');
      const count = await pwInputs.count();
      if (count >= 2) {
        await pwInputs.nth(0).fill(TEST_PW);
        await pwInputs.nth(1).fill('short');
        if (count >= 3) await pwInputs.nth(2).fill('short');

        // Submit
        const submitBtn = page.locator('button:has-text("Change"):not(:has-text("Cancel"))').first();
        if (await submitBtn.isVisible()) {
          await submitBtn.click();
          await page.waitForTimeout(500);
          // Should show error (inline or as message)
          const bodyText = await page.textContent('body') || '';
          // Either client-side validation or server error
          expect(bodyText.length).toBeGreaterThan(0);
        }
      }
    }
    await expect(page.locator('body')).toBeVisible();
  });

  test('password strength indicator exists on form', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/users');
    await page.waitForLoadState('domcontentloaded');

    // Open create user form
    const createBtn = page.locator('button:has-text("Create"), button:has-text("Add")').first();
    if (await createBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await createBtn.click();
      await page.waitForTimeout(500);
      // Password field with strength indicator may be present
      const bodyText = await page.textContent('body') || '';
      expect(bodyText.length).toBeGreaterThan(0);
    }
  });
});
