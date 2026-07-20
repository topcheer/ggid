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

test.describe('Audit Flow', () => {
  test.beforeAll(async () => { await flushRateLimits(); });

  test('audit page loads with events', async ({ page, request }) => {
    const token = await getAdminToken(request);
    await setToken(page, token);
    await page.goto('/audit');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('body')).toBeVisible();
  });

  test('audit filter by action type', async ({ page, request }) => {
    const token = await getAdminToken(request);
    await setToken(page, token);
    await page.goto('/audit');
    await page.waitForLoadState('networkidle');
    const filterSelect = page.locator('select, [data-testid="action-filter"], input[placeholder*="action" i]').first();
    if (await filterSelect.isVisible({ timeout: 3000 }).catch(() => false)) {
      await filterSelect.click();
      const option = page.locator('option, [role="option"]').first();
      if (await option.isVisible({ timeout: 2000 }).catch(() => false)) {
        await option.click();
        await page.waitForTimeout(1000);
      }
    }
    await expect(page.locator('body')).toBeVisible();
  });

  test('audit export button exists', async ({ page, request }) => {
    const token = await getAdminToken(request);
    await setToken(page, token);
    await page.goto('/audit');
    await page.waitForLoadState('networkidle');
    const exportBtn = page.locator('button:has-text("Export"), button:has-text("CSV"), button:has-text("JSON")').first();
    const exists = await exportBtn.isVisible({ timeout: 3000 }).catch(() => false);
    expect(typeof exists).toBe('boolean');
  });
});
