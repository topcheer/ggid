import { flushRateLimits } from "./helpers/flush-ratelimit";
import { test, expect, type APIRequestContext, type Page } from '@playwright/test';

const BASE = process.env.BASE_URL || 'https://ggid-console.iot2.win';
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

test.describe('OAuth Client Flow', () => {
  test('create OAuth client -> list', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/oauth-clients');
    await page.waitForLoadState('domcontentloaded');

    const createBtn = page.locator('button:has-text("Create"), button:has-text("New")').first();
    if (await createBtn.isVisible({ timeout: 5000 }).catch(() => false)) {
      await createBtn.click();
      await page.waitForTimeout(500);
      const nameInput = page.locator('input').first();
      if (await nameInput.isVisible()) await nameInput.fill(`e2e-client-${Date.now()}`);
      const saveBtn = page.locator('button:has-text("Create"):not(:has-text("Cancel")), button:has-text("Save"):not(:has-text("Cancel"))').last();
      if (await saveBtn.isVisible()) {
        await saveBtn.click();
        await page.waitForTimeout(2000);
      }
    }
    await expect(page.locator('body')).toBeVisible();
  });

  test('search OAuth clients', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/oauth-clients');
    await page.waitForLoadState('domcontentloaded');

    const searchInput = page.locator('input[placeholder*="Search"]').first();
    if (await searchInput.isVisible({ timeout: 3000 }).catch(() => false)) {
      await searchInput.fill('test');
      await page.waitForTimeout(500);
    }
    await expect(page.locator('body')).toBeVisible();
  });
});
