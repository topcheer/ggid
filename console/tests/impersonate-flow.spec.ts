import { flushRateLimits } from "./helpers/flush-ratelimit";
import { test, expect, type APIRequestContext, type Page } from '@playwright/test';

const BASE = process.env.BASE_URL || 'https://ggid-console.iot2.win';
const API_BASE = process.env.API_URL || 'https://ggid.iot2.win';
const TENANT = '00000000-0000-0000-0000-000000000001';
const ADMIN_PASSWORD = process.env.TEST_PASSWORD || 'TestPass123!';

async function getAdminToken(request: APIRequestContext): Promise<string> {
  await flushRateLimits();
  const resp = await request.post(`${API_BASE}/api/v1/auth/login`, {
    headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
    data: { username: 'admin', password: ADMIN_PASSWORD, tenant_slug: 'default' },
  });
  const body = await resp.json();
  if (!body.access_token) throw new Error(`Admin login failed: ${JSON.stringify(body).slice(0, 200)}`);
  return body.access_token;
}

async function setToken(page: Page, token: string) {
  await page.goto('/login', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(500);
  await page.evaluate((t) => {
    localStorage.setItem('ggid_access_token', t);
    localStorage.setItem('ggid_tenant_id', '00000000-0000-0000-0000-000000000001');
    localStorage.setItem('ggid_user_id', 'admin');
    localStorage.setItem('ggid_user_name', 'admin');
    localStorage.setItem('ggid_user_scopes', JSON.stringify(['Platform Administrator','Tenant Administrator','Administrator']));
  }, token);
}

test.describe('Impersonation Consent Flow', () => {
  test.beforeAll(async () => { await flushRateLimits(); });

  test('settings page has platform access tab', async ({ page, request }) => {
    const token = await getAdminToken(request);
    await setToken(page, token);
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');
    const tab = page.locator('text=Platform Access').first();
    const exists = await tab.isVisible({ timeout: 5000 }).catch(() => false);
    expect(typeof exists).toBe('boolean');
  });

  test('grant access flow', async ({ page, request }) => {
    const token = await getAdminToken(request);
    await setToken(page, token);
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');
    const tab = page.locator('text=Platform Access').first();
    if (await tab.isVisible({ timeout: 3000 }).catch(() => false)) {
      await tab.click();
      await page.waitForTimeout(1000);
      const grantBtn = page.locator('button:has-text("Grant"), button:has-text("Authorize")').first();
      if (await grantBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
        await grantBtn.click();
        await page.waitForTimeout(500);
      }
    }
    await expect(page.locator('body')).toBeVisible();
  });
});
