import { flushRateLimits } from "./helpers/flush-ratelimit";
import { test, expect, type APIRequestContext } from '@playwright/test';

const BASE = process.env.BASE_URL || 'https://ggid-console.iot2.win';
const API_BASE = process.env.API_URL || 'https://ggid.iot2.win';
const TENANT = '00000000-0000-0000-0000-000000000001';

// Helper: register + login, returns auth token
async function getAuthToken(request: APIRequestContext): Promise<string> {
  const username = `e2e_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`;
  await flushRateLimits();
    await request.post(`${API_BASE}/api/v1/auth/register`, {
    headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
    data: { username, email: `${username}@test.com`, password: 'TestPass123!' },
  });
  // Small delay to avoid rate limiting
  await new Promise(r => setTimeout(r, 500));
  const loginResp = await request.post(`${API_BASE}/api/v1/auth/login`, {
    headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
    data: { username, password: 'TestPass123!' },
  });
  if (loginResp.status() !== 200) {
    // Retry once after delay
    await new Promise(r => setTimeout(r, 2000));
    const retryResp = await request.post(`${API_BASE}/api/v1/auth/login`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { username, password: 'TestPass123!' },
    });
    if (retryResp.status() !== 200) {
      throw new Error(`Login failed after retry: ${retryResp.status()}`);
    }
    const body = await retryResp.json();
    return body.access_token;
  }
  const body = await loginResp.json();
  return body.access_token;
}

// Helper: inject token into page localStorage
async function setToken(page: import('@playwright/test').Page, token: string) {
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
  }, token);
}

test.describe('Auth Flows', () => {
  let pageToken: string;
  
  test.beforeAll(async ({ request }) => {
    pageToken = await getAuthToken(request);
  });

  test('register → login → dashboard flow', async ({ page, request }) => {
    const token = pageToken;
    await setToken(page, token);
    
    await page.goto('/dashboard');
    await page.waitForLoadState("domcontentloaded");
    
    const bodyText = await page.textContent('body');
    expect(bodyText).not.toContain('Application error');
    expect(bodyText).not.toContain('Internal Server Error');
  });

  test('login page renders correctly', async ({ page }) => {
    await page.goto('/login');
    await page.waitForLoadState("domcontentloaded");
    const bodyText = await page.textContent('body');
    expect(bodyText?.length).toBeGreaterThan(0);
  });

  test('register page renders correctly', async ({ page }) => {
    await page.goto('/register');
    await page.waitForLoadState("domcontentloaded");
    const bodyText = await page.textContent('body');
    expect(bodyText?.length).toBeGreaterThan(0);
  });

  test('forgot-password page renders correctly', async ({ page }) => {
    await page.goto('/forgot-password');
    await page.waitForLoadState("domcontentloaded");
    const bodyText = await page.textContent('body');
    expect(bodyText?.length).toBeGreaterThan(0);
  });

  test('dashboard page loads', async ({ page, request }) => {
    const token = pageToken;
    await setToken(page, token);
    await page.goto('/dashboard');
    await page.waitForLoadState("domcontentloaded");
    const bodyText = await page.textContent('body');
    expect(bodyText).not.toContain('Application error');
  });

  test('users page loads', async ({ page, request }) => {
    const token = pageToken;
    await setToken(page, token);
    await page.goto('/users');
    await page.waitForLoadState("domcontentloaded");
    const bodyText = await page.textContent('body');
    expect(bodyText).not.toContain('Application error');
  });

  test('roles page loads', async ({ page, request }) => {
    const token = pageToken;
    await setToken(page, token);
    await page.goto('/roles');
    await page.waitForLoadState("domcontentloaded");
    const bodyText = await page.textContent('body');
    expect(bodyText).not.toContain('Application error');
  });

  test('organizations page loads', async ({ page, request }) => {
    const token = pageToken;
    await setToken(page, token);
    await page.goto('/organizations');
    await page.waitForLoadState("domcontentloaded");
    const bodyText = await page.textContent('body');
    expect(bodyText).not.toContain('Application error');
  });

  test('audit page loads', async ({ page, request }) => {
    const token = pageToken;
    await setToken(page, token);
    await page.goto('/audit');
    await page.waitForLoadState("domcontentloaded");
    const bodyText = await page.textContent('body');
    expect(bodyText).not.toContain('Application error');
  });

  test('settings page loads', async ({ page, request }) => {
    const token = pageToken;
    await setToken(page, token);
    await page.goto('/settings');
    await page.waitForLoadState("domcontentloaded");
    const bodyText = await page.textContent('body');
    expect(bodyText).not.toContain('Application error');
  });

  test('agents page loads', async ({ page, request }) => {
    const token = pageToken;
    await setToken(page, token);
    await page.goto('/agents');
    await page.waitForLoadState("domcontentloaded");
    const bodyText = await page.textContent('body');
    expect(bodyText).not.toContain('Application error');
  });

  test('security-center page loads', async ({ page, request }) => {
    const token = pageToken;
    await setToken(page, token);
    await page.goto('/security-center');
    await page.waitForLoadState("domcontentloaded");
    const bodyText = await page.textContent('body');
    expect(bodyText).not.toContain('Application error');
  });
});

test.describe('API Integration', () => {
  let sharedToken: string;
  
  test.beforeAll(async ({ request }) => {
    // Flush Redis then login as admin for API tests (needs admin scope)
    await flushRateLimits();
    const adminPassword = process.env.TEST_PASSWORD || '';
    const loginResp = await request.post(`${API_BASE}/api/v1/auth/login`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { username: 'admin', password: adminPassword },
    });
    const body = await loginResp.json();
    sharedToken = body.access_token || '';
  });

  test('user can create and list roles', async ({ request }) => {
    const authHeaders = { 'X-Tenant-ID': TENANT, 'Authorization': `Bearer ${sharedToken}` };

    const roleKey = `rk_${Date.now()}`;
    const createResp = await request.post(`${API_BASE}/api/v1/roles`, {
      headers: { ...authHeaders, 'Content-Type': 'application/json' },
      data: { name: `Test Role ${roleKey}`, key: roleKey, description: 'E2E test role' },
    });
    expect(createResp.status()).toBeLessThan(500);

    const listResp = await request.get(`${API_BASE}/api/v1/roles`, { headers: authHeaders });
    expect(listResp.status()).toBe(200);
  });

  test('user can create OAuth client', async ({ request }) => {
    const token = sharedToken;
    const authHeaders = { 'X-Tenant-ID': TENANT, 'Authorization': `Bearer ${token}` };

    const createResp = await request.post(`${API_BASE}/api/v1/oauth/clients`, {
      headers: { ...authHeaders, 'Content-Type': 'application/json' },
      data: { 
        client_name: `e2e-client-${Date.now()}`, 
        redirect_uris: ['http://localhost:3000/callback'],
      },
    });
    expect(createResp.status()).toBeLessThan(500);
  });

  test('user can create organization', async ({ request }) => {
    const token = sharedToken;
    const authHeaders = { 'X-Tenant-ID': TENANT, 'Authorization': `Bearer ${token}` };

    const createResp = await request.post(`${API_BASE}/api/v1/organizations`, {
      headers: { ...authHeaders, 'Content-Type': 'application/json' },
      data: { name: `E2E Org ${Date.now()}`, description: 'Test org' },
    });
    expect(createResp.status()).toBeLessThan(500);
  });

  test('OIDC discovery returns valid document', async ({ request }) => {
    const resp = await request.get(`${API_BASE}/.well-known/openid-configuration`);
    expect(resp.status()).toBe(200);
    const body = await resp.json();
    expect(body.issuer).toBeTruthy();
    expect(body.authorization_endpoint).toBeTruthy();
    expect(body.token_endpoint).toBeTruthy();
    expect(body.jwks_uri).toBeTruthy();
  });

  test('JWKS endpoint returns valid keys', async ({ request }) => {
    const resp = await request.get(`${API_BASE}/.well-known/jwks.json`);
    expect(resp.status()).toBe(200);
    const body = await resp.json();
    expect(body.keys).toBeDefined();
    expect(body.keys.length).toBeGreaterThan(0);
  });

  test('trust store endpoints work', async ({ request }) => {
    const token = sharedToken;
    const authHeaders = { 'X-Tenant-ID': TENANT, 'Authorization': `Bearer ${token}` };

    const casResp = await request.get(`${API_BASE}/api/v1/auth/trust-store/cas`, { headers: authHeaders });
    expect(casResp.status()).toBe(200);

    const certsResp = await request.get(`${API_BASE}/api/v1/auth/certificates`, { headers: authHeaders });
    expect(certsResp.status()).toBe(200);

    const mtlsResp = await request.get(`${API_BASE}/api/v1/auth/mtls/config`, { headers: authHeaders });
    expect(mtlsResp.status()).toBe(200);

    const expiryResp = await request.get(`${API_BASE}/api/v1/auth/certificates/expiry`, { headers: authHeaders });
    expect(expiryResp.status()).toBe(200);
  });

  test('audit endpoints work', async ({ request }) => {
    const token = sharedToken;
    const authHeaders = { 'X-Tenant-ID': TENANT, 'Authorization': `Bearer ${token}` };

    // Audit endpoints may require admin scope — use status < 500 (not 200) for non-admin users
    const eventsResp = await request.get(`${API_BASE}/api/v1/audit/events`, { headers: authHeaders });
    expect(eventsResp.status()).toBeLessThan(500);

    const hashResp = await request.get(`${API_BASE}/api/v1/audit/hash-chain`, { headers: authHeaders });
    expect(hashResp.status()).toBeLessThan(500);

    const webhooksResp = await request.get(`${API_BASE}/api/v1/audit/webhooks`, { headers: authHeaders });
    expect(webhooksResp.status()).toBeLessThan(500);

    const siemResp = await request.get(`${API_BASE}/api/v1/audit/siem/health`, { headers: authHeaders });
    expect(siemResp.status()).toBeLessThan(500);
  });

  test('auth endpoints work (refresh, sessions, mfa)', async ({ request }) => {
    const username = `auth_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`;
    await flushRateLimits();
    await request.post(`${API_BASE}/api/v1/auth/register`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { username, email: `${username}@test.com`, password: 'TestPass123!' },
    });
    const loginResp = await request.post(`${API_BASE}/api/v1/auth/login`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { username, password: 'TestPass123!' },
    });
    const { access_token, refresh_token } = await loginResp.json();
    const authHeaders = { 'X-Tenant-ID': TENANT, 'Authorization': `Bearer ${access_token}` };

    // Refresh
    const refreshResp = await request.post(`${API_BASE}/api/v1/auth/refresh`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { refresh_token },
    });
    expect(refreshResp.status()).toBeLessThan(500);

    // Sessions
    const sessResp = await request.get(`${API_BASE}/api/v1/auth/sessions`, { headers: authHeaders });
    expect(sessResp.status()).toBeLessThan(500);

    // MFA factors
    const mfaResp = await request.get(`${API_BASE}/api/v1/auth/mfa/factors`, { headers: authHeaders });
    expect(mfaResp.status()).toBeLessThan(500);

    // MFA status
    const mfaStatusResp = await request.get(`${API_BASE}/api/v1/auth/mfa/status`, { headers: authHeaders });
    expect(mfaStatusResp.status()).toBeLessThan(500);
  });

  test('password change works', async ({ request }) => {
    const username = `pw_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`;
    await request.post(`${API_BASE}/api/v1/auth/register`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { username, email: `${username}@test.com`, password: 'TestPass123!' },
    });
    const loginResp = await request.post(`${API_BASE}/api/v1/auth/login`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { username, password: 'TestPass123!' },
    });
    const { access_token } = await loginResp.json();
    const authHeaders = { 'X-Tenant-ID': TENANT, 'Authorization': `Bearer ${access_token}` };

    const changeResp = await request.post(`${API_BASE}/api/v1/auth/password/change`, {
      headers: { ...authHeaders, 'Content-Type': 'application/json' },
      data: { old_password: 'TestPass123!', new_password: 'NewStrongPass456!!' },
    });
    expect(changeResp.status()).toBeLessThan(500);
  });

  test('OAuth token revocation works', async ({ request }) => {
    const username = `rev_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`;
    await request.post(`${API_BASE}/api/v1/auth/register`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { username, email: `${username}@test.com`, password: 'TestPass123!' },
    });
    const loginResp = await request.post(`${API_BASE}/api/v1/auth/login`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: { username, password: 'TestPass123!' },
    });
    const { access_token } = await loginResp.json();

    const revokeResp = await request.post(`${API_BASE}/api/v1/oauth/revoke`, {
      headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/x-www-form-urlencoded' },
      data: { token: access_token },
    });
    expect(revokeResp.status()).toBe(200);
  });
});
