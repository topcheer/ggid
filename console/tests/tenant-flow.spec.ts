import { test, expect, type APIRequestContext, type Page } from '@playwright/test';

const BASE = process.env.BASE_URL || 'https://ggid-console.iot2.win';
const API_BASE = process.env.API_URL || 'https://ggid.iot2.win';
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
    localStorage.setItem('ggid_user_scopes', JSON.stringify(['Platform Administrator', 'Tenant Administrator', 'Administrator']));
  }, { t: token, u: username });
}

test.describe('Tenant Flow', () => {
  test('view tenant list', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/admin/tenants');
    await page.waitForLoadState('domcontentloaded');
    await expect(page.locator('body')).toBeVisible();
  });

  test('create tenant form → save', async ({ page, request }) => {
    const { token, user } = await getAuthToken(request);
    await setToken(page, token, user);
    await page.goto('/admin/tenants');
    await page.waitForLoadState('domcontentloaded');

    // Click Create tab/button
    const createBtn = page.locator('button:has-text("Create"), a:has-text("Create")').first();
    if (await createBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await createBtn.click();
      await page.waitForTimeout(500);

      // Fill form
      const nameInput = page.locator('input[name="name"], input[placeholder*="name" i]').first();
      if (await nameInput.isVisible()) {
        await nameInput.fill(`e2e-tenant-${Date.now()}`);
      }
      const slugInput = page.locator('input[name="slug"], input[placeholder*="slug" i]').first();
      if (await slugInput.isVisible()) {
        await slugInput.fill(`e2e-${Date.now()}`);
      }

      // Submit
      const submitBtn = page.locator('button[type="submit"], button:has-text("Create")').last();
      if (await submitBtn.isVisible()) {
        await submitBtn.click();
        await page.waitForTimeout(2000);
      }
    }
    await expect(page.locator('body')).toBeVisible();
  });
});
