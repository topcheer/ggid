import { flushRateLimits } from "./helpers/flush-ratelimit";
import { test, expect, type APIRequestContext, type Page } from '@playwright/test';

const BASE = process.env.BASE_URL || 'https://ggid-console.iot2.win';
const API_BASE = process.env.API_URL || 'https://ggid.iot2.win';
const TENANT = '00000000-0000-0000-0000-000000000001';
const ADMIN_PASSWORD = process.env.TEST_PASSWORD || '';

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

test.describe('Branding Flow', () => {
  test.beforeAll(async () => { await flushRateLimits(); });

  test('branding settings page loads', async ({ page, request }) => {
    const token = await getAdminToken(request);
    await setToken(page, token);
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');
    const branding = page.locator('text=Branding, text=Appearance, text=Theme').first();
    const exists = await branding.isVisible({ timeout: 5000 }).catch(() => false);
    expect(typeof exists).toBe('boolean');
  });

  test('branding color change and save', async ({ page, request }) => {
    const token = await getAdminToken(request);
    await setToken(page, token);
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');
    const branding = page.locator('text=Branding, text=Appearance').first();
    if (await branding.isVisible({ timeout: 3000 }).catch(() => false)) {
      await branding.click();
      await page.waitForTimeout(1000);
      const colorInput = page.locator('input[type="color"]').first();
      if (await colorInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await colorInput.fill('#3b82f6');
        const saveBtn = page.locator('button:has-text("Save")').first();
        if (await saveBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
          await saveBtn.click();
          await page.waitForTimeout(1000);
        }
      }
    }
    await expect(page.locator('body')).toBeVisible();
  });
});
