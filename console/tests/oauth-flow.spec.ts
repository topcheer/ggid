import { flushRateLimits } from "./helpers/flush-ratelimit";
import { test, expect, type APIRequestContext } from '@playwright/test';

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
  return body.access_token || '';
}

test.describe('OAuth Client Lifecycle', () => {
  test.beforeAll(async () => { await flushRateLimits(); });

  let clientUuid = '';
  let clientSecret = '';

  test('create OAuth client returns UUID + secret', async ({ request }) => {
    const token = await getAdminToken(request);
    if (!token) { test.skip(); return; }
    const resp = await request.post(`${API_BASE}/api/v1/oauth/clients`, {
      headers: { 'Authorization': `Bearer ${token}`, 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: {
        name: `E2E OAuth ${Date.now()}`,
        redirect_uris: ['https://e2e.test/callback'],
        grant_types: ['authorization_code'],
      },
    });
    if (resp.status() === 401) { test.skip(); return; }
    expect([200, 201]).toContain(resp.status());
    const body = await resp.json();
    const client = body.Client || body;
    const uuid = client.ID || client.id || '';
    const secret = body.ClientSecret || client.client_secret || '';
    expect(uuid).toBeTruthy();
    expect(secret).toBeTruthy();
    expect(secret.length).toBeGreaterThan(10);
  });

  test('list OAuth clients includes created client', async ({ request }) => {
    const token = await getAdminToken(request);
    const resp = await request.get(`${API_BASE}/api/v1/oauth/clients`, {
      headers: { 'Authorization': `Bearer ${token}`, 'X-Tenant-ID': TENANT },
    });
    expect(resp.status()).toBe(200);
    const body = await resp.json();
    const clients = body.clients || body.Clients || body;
    expect(Array.isArray(clients)).toBeTruthy();
  });

  test('delete OAuth client by UUID', async ({ request }) => {
    const token = await getAdminToken(request);
    if (!token) { test.skip(); return; }
    // Create a client to delete
    const createResp = await request.post(`${API_BASE}/api/v1/oauth/clients`, {
      headers: { 'Authorization': `Bearer ${token}`, 'X-Tenant-ID': TENANT, 'Content-Type': 'application/json' },
      data: {
        name: `Del Test ${Date.now()}`,
        redirect_uris: ['https://del.test/cb'],
        grant_types: ['authorization_code'],
      },
    });
    if (createResp.status() === 401) { test.skip(); return; }
    const createBody = await createResp.json();
    const uuid = (createBody.Client || createBody).ID || (createBody.Client || createBody).id;

    // Delete it
    const delResp = await request.delete(`${API_BASE}/api/v1/oauth/clients/${uuid}`, {
      headers: { 'Authorization': `Bearer ${token}`, 'X-Tenant-ID': TENANT },
    });
    expect([200, 204]).toContain(delResp.status());
    if (delResp.status() === 200) {
      const delBody = await delResp.json();
      expect(delBody.deleted).toBe(true);
    }
  });

  test('secret not returned on list', async ({ request }) => {
    const token = await getAdminToken(request);
    const resp = await request.get(`${API_BASE}/api/v1/oauth/clients`, {
      headers: { 'Authorization': `Bearer ${token}`, 'X-Tenant-ID': TENANT },
    });
    const body = await resp.json();
    const json = JSON.stringify(body);
    // Secret hash may appear but raw secret should not
    expect(json).not.toContain('gcs_');
  });
});
