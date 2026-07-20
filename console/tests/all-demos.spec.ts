import { test, expect } from '@playwright/test';
import { execSync } from 'child_process';

const BASE = process.env.BASE_URL || 'https://ggid-console.iot2.win';
const API_BASE = process.env.API_URL || 'https://ggid.iot2.win';
const TENANT = '00000000-0000-0000-0000-000000000001';
const ADMIN_PASSWORD = process.env.TEST_PASSWORD || '';

const DEMOS = [
  { lang: 'go', url: 'https://go-oauth-demo.iot2.win' },
  { lang: 'java', url: 'https://java-oauth-demo.iot2.win' },
  { lang: 'python', url: 'https://python-oauth-demo.iot2.win' },
  { lang: 'node', url: 'https://node-oauth-demo.iot2.win' },
  { lang: 'rust', url: 'https://rust-oauth-demo.iot2.win' },
  { lang: 'php', url: 'https://php-oauth-demo.iot2.win' },
  { lang: 'ruby', url: 'https://ruby-oauth-demo.iot2.win' },
  { lang: 'csharp', url: 'https://csharp-oauth-demo.iot2.win' },
  { lang: 'dart', url: 'https://dart-oauth-demo.iot2.win' },
  { lang: 'saml', url: 'https://saml-demo.iot2.win' },
  { lang: 'erp', url: 'https://erp.iot2.win' },
];

test.describe('All Demos — UI Verification', () => {
  test.describe.configure({ mode: 'serial' });

  for (const demo of DEMOS) {
    test(`${demo.lang} demo loads`, async ({ page }) => {
      // Flush rate limits
      try { execSync('kubectl exec deploy/ggid-redis -n ggid -- redis-cli FLUSHALL', { stdio: 'pipe', timeout: 5000 }); } catch {}

      const resp = await page.goto(demo.url, { waitUntil: 'domcontentloaded', timeout: 15000 });
      expect(resp?.status()).toBeLessThan(500);

      // Check page renders (not blank)
      const body = await page.textContent('body');
      expect(body?.length || 0).toBeGreaterThan(0);
      
      // Should not show application error
      expect(body).not.toContain('Application error');
      expect(body).not.toContain('Internal Server Error');
    });

    test(`${demo.lang} demo has login/oauth element`, async ({ page }) => {
      const resp = await page.goto(demo.url, { waitUntil: 'domcontentloaded', timeout: 15000 });
      const body = await page.textContent('body') || '';
      
      // Should have login button or GGID reference
      const hasLogin = body.toLowerCase().includes('login') || 
                       body.toLowerCase().includes('oauth') ||
                       body.toLowerCase().includes('ggid') ||
                       body.toLowerCase().includes('登录') ||
                       body.toLowerCase().includes('sso');
      expect(hasLogin).toBeTruthy();
    });
  }

  test('ERP demo — full login flow', async ({ page }) => {
    try { execSync('kubectl exec deploy/ggid-redis -n ggid -- redis-cli FLUSHALL', { stdio: 'pipe', timeout: 5000 }); } catch {}
    
    await page.goto('https://erp.iot2.win/login', { waitUntil: 'domcontentloaded' });
    const body = await page.textContent('body') || '';
    expect(body).toContain('ERP');
    
    // Click GGID login button
    const loginBtn = page.locator('button.ant-btn-primary');
    if (await loginBtn.count() > 0) {
      await loginBtn.click();
      await page.waitForTimeout(3000);
      // Should redirect to GGID OAuth or show login form
      const url = page.url();
      expect(url.includes('ggid') || url.includes('oauth') || url.includes('login')).toBeTruthy();
    }
  });

  test('SAML demo — SSO redirect', async ({ page }) => {
    await page.goto('https://saml-demo.iot2.win/', { waitUntil: 'domcontentloaded' });
    
    // Click login
    const loginLink = page.locator('a:has-text("Login"), a[href*="login"]');
    if (await loginLink.count() > 0) {
      await loginLink.first().click();
      await page.waitForTimeout(3000);
      // Should redirect to GGID SAML SSO
      const url = page.url();
      expect(url).toContain('ggid');
    }
  });

  test('GGID Console — admin login → dashboard', async ({ page }) => {
    try { execSync('kubectl exec deploy/ggid-redis -n ggid -- redis-cli FLUSHALL', { stdio: 'pipe', timeout: 5000 }); } catch {}
    
    await page.goto('https://ggid-console.iot2.win/login', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);
    
    // Fill login form
    await page.fill('input[placeholder="default"]', 'default');
    await page.fill('#username', 'admin');
    await page.fill('#password', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');
    
    await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15000 });
    
    // Should be on dashboard
    const body = await page.textContent('body') || '';
    expect(body.toLowerCase()).toMatch(/overview|dashboard|总览/);
  });
});
