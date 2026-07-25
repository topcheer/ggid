const { chromium } = require('playwright');

const BASE = 'https://ggid-console.iot2.win';
const TENANT = 'fb44ca98-2a8a-498b-a9b2-00fc014524ce';
const OUT = '/Volumes/new/ggai/ggid/docs/demo/screenshots';

const accounts = {
  admin: { username: 'admin', password: 'SecureAdmin@Pass2026#Xq', scopes: ['platform:admin','admin','tenant:admin','Administrator'] },
  tenant_admin: { username: 'demo_tenant_admin', password: 'DemoAdmin@Pass2026#Xq', scopes: ['admin','tenant:admin','Administrator'] },
  user: { username: 'demo_user', password: 'DemoUser@Pass2026', scopes: ['user:self','viewer','Viewer'] },
};

async function login(page, acct) {
  // Get token via API
  const resp = await page.evaluate(async (params) => {
    const r = await fetch('/api/v1/oauth/token', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded', 'X-Tenant-ID': params.tid },
      body: `grant_type=password&username=${params.username}&password=${params.password}&client_id=gcid-console&scope=admin`
    });
    return r.json();
  }, { ...acct, tid: TENANT });

  if (!resp.access_token) { console.error('Login failed:', acct.username, resp); return false; }

  // Set localStorage
  await page.evaluate((data) => {
    localStorage.setItem('ggid_access_token', data.token);
    localStorage.setItem('ggid_tenant_id', data.tid);
    localStorage.setItem('ggid_user_scopes', JSON.stringify(data.scopes));
    localStorage.setItem('ggid_user_id', data.uid || 'admin');
  }, { token: resp.access_token, tid: TENANT, scopes: acct.scopes, uid: acct.uid });

  // Get user ID
  const me = await page.evaluate(async (params) => {
    const r = await fetch('/api/v1/me', { headers: { 'Authorization': 'Bearer ' + params.token, 'X-Tenant-ID': params.tid } });
    return r.json();
  }, { token: resp.access_token, tid: TENANT });

  if (me.id) {
    await page.evaluate((uid) => localStorage.setItem('ggid_user_id', uid), me.id);
  }

  console.log(`  ✓ ${acct.username} logged in (scope: ${resp.scope || 'N/A'})`);
  return true;
}

async function screenshot(page, url, filename) {
  await page.goto(url, { waitUntil: 'networkidle', timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(2000);
  await page.setViewportSize({ width: 1920, height: 1080 });
  const path = `${OUT}/${filename}`;
  await page.screenshot({ path, fullPage: false });
  console.log(`  📸 ${filename}`);
}

(async () => {
  const browser = await chromium.launch({ headless: true });

  // === Instance Admin ===
  console.log('\n=== Instance Admin ===');
  const adminPage = await browser.newPage();
  await adminPage.goto(BASE + '/login', { waitUntil: 'networkidle' });
  await login(adminPage, accounts.admin);
  
  await screenshot(adminPage, BASE + '/dashboard', '01-admin-dashboard.png');
  await screenshot(adminPage, BASE + '/users', '02-admin-users.png');
  await screenshot(adminPage, BASE + '/roles', '03-admin-roles.png');
  await screenshot(adminPage, BASE + '/organizations', '04-admin-organizations.png');
  await screenshot(adminPage, BASE + '/security/itdr', '05-admin-itdr.png');
  await screenshot(adminPage, BASE + '/settings/conditional-access', '06-admin-cae.png');
  await screenshot(adminPage, BASE + '/security/posture', '07-admin-posture.png');
  await screenshot(adminPage, BASE + '/security/overview', '08-admin-security-overview.png');
  await screenshot(adminPage, BASE + '/audit', '09-admin-audit.png');
  await screenshot(adminPage, BASE + '/oauth-clients', '10-admin-oauth.png');
  await screenshot(adminPage, BASE + '/webhooks', '11-admin-webhooks.png');
  await screenshot(adminPage, BASE + '/api-keys', '12-admin-api-keys.png');
  await screenshot(adminPage, BASE + '/admin/impersonate', '13-admin-impersonate.png');
  await screenshot(adminPage, BASE + '/admin/secrets', '14-admin-secrets.png');
  await screenshot(adminPage, BASE + '/admin/backup', '15-admin-backup.png');
  await screenshot(adminPage, BASE + '/admin/tenants', '16-admin-tenants.png');
  await screenshot(adminPage, BASE + '/admin/audit/global', '17-admin-global-audit.png');
  await screenshot(adminPage, BASE + '/admin/threats', '18-admin-threats.png');
  await screenshot(adminPage, BASE + '/settings/mfa', '19-admin-mfa.png');
  await screenshot(adminPage, BASE + '/settings/branding', '20-admin-branding.png');
  await screenshot(adminPage, BASE + '/settings/feature-flags', '21-admin-feature-flags.png');
  await screenshot(adminPage, BASE + '/settings/saml-config', '22-admin-saml.png');
  await screenshot(adminPage, BASE + '/settings/password-policy', '23-admin-password-policy.png');
  await screenshot(adminPage, BASE + '/sessions', '24-admin-sessions.png');

  // === Tenant Admin ===
  console.log('\n=== Tenant Admin ===');
  const taPage = await browser.newPage();
  await taPage.goto(BASE + '/login', { waitUntil: 'networkidle' });
  await login(taPage, accounts.tenant_admin);
  
  await screenshot(taPage, BASE + '/dashboard', '25-tenant-admin-dashboard.png');
  await screenshot(taPage, BASE + '/users', '26-tenant-admin-users.png');
  await screenshot(taPage, BASE + '/admin/impersonate', '27-tenant-admin-impersonate.png');

  // === Regular User ===
  console.log('\n=== Regular User ===');
  const userPage = await browser.newPage();
  await userPage.goto(BASE + '/login', { waitUntil: 'networkidle' });
  await login(userPage, accounts.user);
  
  await screenshot(userPage, BASE + '/dashboard', '28-user-dashboard.png');
  await screenshot(userPage, BASE + '/sessions', '29-user-sessions.png');
  await screenshot(userPage, BASE + '/access-requests', '30-user-access-requests.png');

  await browser.close();
  console.log('\n✅ All screenshots complete!');
})();
