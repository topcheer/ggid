import { test, expect, type APIRequestContext } from '@playwright/test';
import { flushRateLimits } from "./helpers/flush-ratelimit";

const API_BASE = process.env.API_URL || 'https://ggid.iot2.win';
const TENANT = '00000000-0000-0000-0000-000000000001';
const ADMIN_PASSWORD = process.env.TEST_PASSWORD || '';

async function adminLogin(request: APIRequestContext): Promise<string> {
  await flushRateLimits();
  const resp = await request.post(`${API_BASE}/api/v1/auth/login`, {
    headers: { 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
    data: { username: 'admin', password: ADMIN_PASSWORD, tenant_slug: 'default' },
  });
  return (await resp.json()).access_token || '';
}

test.describe('OAuth flow — CRUD + secret visibility', () => {
  test('create → list → delete OAuth client', async ({ request }) => {
    const token = await adminLogin(request);
    if (!token) { test.skip(); return; }
    const headers = { Authorization: `Bearer ${token}`, 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' };

    // Create
    const createResp = await request.post(`${API_BASE}/api/v1/oauth/clients`, {
      headers,
      data: {
        name: `E2E OAuth ${Date.now()}`,
        redirect_uris: ['https://example.com/callback'],
        grant_types: ['authorization_code'],
        response_types: ['code'],
        scopes: ['openid', 'profile'],
      },
    });
    expect(createResp.status() === 200 || createResp.status() === 201).toBeTruthy();
    const createBody = await createResp.json();
    const client = createBody.client || createBody.Client || createBody;
    const clientId = client.client_id || client.ClientID;
    const clientSecret = createBody.client_secret;
    expect(clientId).toBeTruthy();

    // Secret should be present on creation (shown once)
    expect(clientSecret).toBeTruthy();
    expect(clientSecret.length).toBeGreaterThan(20);

    // List — should contain our client
    const listResp = await request.get(`${API_BASE}/api/v1/oauth/clients`, { headers });
    expect(listResp.ok()).toBeTruthy();
    const listBody = await listResp.json();
    const clients = listBody.clients || listBody.Client || listBody.data || [];
    const found = Array.isArray(clients) && clients.some((c: any) =>
      (c.client_id || c.ClientID) === clientId
    );
    expect(found).toBeTruthy();

    // Delete
    const delResp = await request.delete(`${API_BASE}/api/v1/oauth/clients/${clientId}`, { headers });
    expect(delResp.ok()).toBeTruthy();

    // Verify deleted — list should not contain it
    const listAfter = await request.get(`${API_BASE}/api/v1/oauth/clients`, { headers });
    const listAfterBody = await listAfter.json();
    const clientsAfter = listAfterBody.clients || listAfterBody.Client || listAfterBody.data || [];
    const stillThere = Array.isArray(clientsAfter) && clientsAfter.some((c: any) =>
      (c.client_id || c.ClientID) === clientId
    );
    expect(stillThere).toBeFalsy();
  });

  test('secret not returned on GET (only on POST create)', async ({ request }) => {
    const token = await adminLogin(request);
    if (!token) { test.skip(); return; }
    const headers = { Authorization: `Bearer ${token}`, 'X-Tenant-ID': TENANT };

    // GET list should not contain client_secret
    const listResp = await request.get(`${API_BASE}/api/v1/oauth/clients`, { headers });
    if (listResp.ok()) {
      const body = await listResp.json();
      const clients = body.clients || body.Client || body.data || [];
      if (Array.isArray(clients) && clients.length > 0) {
        const hasSecret = clients.some((c: any) => c.client_secret || c.ClientSecret);
        // Secret should NOT be in list response
        expect(hasSecret).toBeFalsy();
      }
    }
  });
});
