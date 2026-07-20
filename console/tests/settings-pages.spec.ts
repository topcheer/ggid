import { flushRateLimits } from "./helpers/flush-ratelimit";
import { test, expect } from '@playwright/test';
import { PAGE_CATEGORIES } from './pages-data';

const API_BASE = process.env.API_URL || 'https://ggid.iot2.win';
const TENANT = '00000000-0000-0000-0000-000000000001';

// Test all settings pages load without errors
test.describe.configure({ mode: 'serial' });

const settingsPages = PAGE_CATEGORIES['settings'] || [];

for (const pagePath of settingsPages) {
  test(`settings page loads: ${pagePath}`, async ({ page, request }) => {
    // First login via API to get token
    const username = `set_${Math.random().toString(36).slice(2, 10)}`;
    await flushRateLimits();
    await request.post(`${API_BASE}/api/v1/auth/register`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { username, email: `${username}@test.com`, password: 'TestPass123!' },
    });
    const loginResp = await request.post(`${API_BASE}/api/v1/auth/login`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { username, password: 'TestPass123!' },
    });
    const { access_token } = await loginResp.json();
    
    // Set token and visit page
    await page.goto('/login');
    await page.evaluate((t) => {
      localStorage.setItem('ggid_access_token', t);
    localStorage.setItem('ggid_tenant_id', '00000000-0000-0000-0000-000000000001');
    localStorage.setItem('ggid_user_id', 'admin');
    localStorage.setItem('ggid_user_name', 'admin');
    localStorage.setItem('ggid_user_scopes', JSON.stringify(['Platform Administrator','Tenant Administrator','Administrator']));
      localStorage.setItem('ggid_access_token', t);
    localStorage.setItem('ggid_tenant_id', '00000000-0000-0000-0000-000000000001');
    localStorage.setItem('ggid_user_id', 'admin');
    localStorage.setItem('ggid_user_name', 'admin');
    localStorage.setItem('ggid_user_scopes', JSON.stringify(['Platform Administrator','Tenant Administrator','Administrator']));
    }, access_token);
    
    const response = await page.goto(pagePath, { waitUntil: 'networkidle' });
    expect(response?.status()).toBeLessThan(500);
    
    const bodyText = await page.textContent('body');
    expect(bodyText).not.toContain('Application error');
    expect(bodyText).not.toContain('Internal Server Error');
  });
}
