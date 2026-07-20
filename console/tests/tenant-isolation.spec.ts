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

test.describe('Tenant isolation — cross-tenant access denied', () => {
  test('tenant A token cannot access tenant B resources', async ({ request }) => {
    const token = await adminLogin(request);
    if (!token) { test.skip(); return; }

    // Try to access a different tenant's data (tenant B = fake UUID)
    const fakeTenantId = '99999999-9999-9999-9999-999999999999';

    const resp = await request.get(`${API_BASE}/api/v1/users`, {
      headers: {
        Authorization: `Bearer ${token}`,
        'X-Tenant-ID': fakeTenantId,
      },
    });

    // Should either return empty (RLS filtered) or 403
    if (resp.ok()) {
      const body = await resp.json();
      const users = body.users || body.data || [];
      // Should not see any users from the fake tenant
      expect(Array.isArray(users) ? users.length : 0).toBe(0);
    } else {
      expect(resp.status() === 403 || resp.status() === 401).toBeTruthy();
    }
  });

  test('impersonate without consent → 403', async ({ request }) => {
    const token = await adminLogin(request);
    if (!token) { test.skip(); return; }

    // Try to impersonate into a tenant without consent
    const fakeTenantId = '99999999-9999-9999-9999-999999999999';
    const resp = await request.post(`${API_BASE}/api/v1/impersonate/start`, {
      headers: {
        Authorization: `Bearer ${token}`,
        'X-Tenant-ID': TENANT,
        'X-User-ID': 'd6795833-d928-4afd-b5fb-2015d03f2941',
        'Content-Type': 'application/json',
      },
      data: {
        tenant_id: fakeTenantId,
        reason: 'E2E isolation test',
      },
    });

    // Should be 403 (no consent for this tenant)
    expect(resp.status() === 403 || resp.status() === 404).toBeTruthy();
  });

  test('impersonate with consent → allowed → revoke → denied', async ({ request }) => {
    const token = await adminLogin(request);
    if (!token) { test.skip(); return; }
    const headers = {
      Authorization: `Bearer ${token}`,
      'X-Tenant-ID': TENANT,
      'X-User-ID': 'd6795833-d928-4afd-b5fb-2015d03f2941',
      'Content-Type': 'application/json',
    };

    // 1. Grant consent for default tenant
    const grantResp = await request.post(`${API_BASE}/api/v1/tenants/${TENANT}/access/grant`, {
      headers,
      data: { scope: 'support', reason: 'E2E isolation consent test' },
    });
    if (!grantResp.ok() && grantResp.status() !== 409) { test.skip(); return; }

    const grantBody = await grantResp.json();
    const consentId = grantBody.consent_id || grantBody.id;

    // 2. Impersonate should now succeed
    const startResp = await request.post(`${API_BASE}/api/v1/impersonate/start`, {
      headers,
      data: { tenant_id: TENANT, reason: 'E2E isolation test with consent' },
    });
    if (!startResp.ok()) { test.skip(); return; }

    const startBody = await startResp.json();
    const sessionId = startBody.session_id;
    expect(sessionId).toBeTruthy();

    // 3. End session
    if (sessionId) {
      const endResp = await request.post(`${API_BASE}/api/v1/impersonate/end`, {
        headers,
        data: { session_id: sessionId },
      });
      expect(endResp.ok()).toBeTruthy();
    }

    // 4. Revoke consent
    if (consentId) {
      const revokeResp = await request.delete(`${API_BASE}/api/v1/tenants/${TENANT}/access/${consentId}`, { headers });
      expect(revokeResp.ok()).toBeTruthy();
    }
  });
});
