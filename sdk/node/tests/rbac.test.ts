import { describe, it, expect, vi, beforeEach } from 'vitest';
import { GGIDClient } from '../src/client';
import '../src/policy';

function mockFetch(data: any, status = 200) {
  const fetchMock = vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(JSON.stringify(data)),
  });
  (globalThis as any).fetch = fetchMock;
  return fetchMock;
}

describe('RBAC / checkPermission', () => {
  let client: GGIDClient;

  beforeEach(() => {
    client = new GGIDClient({ gatewayUrl: 'https://iam.test.com' });
  });

  it('checkPermission returns allowed=true when policy allows', async () => {
    mockFetch({ allowed: true, reason: 'role has permission' });
    const result = await client.checkPermission('token', 'products', 'read');
    expect(result.allowed).toBe(true);
    expect(result.reason).toBe('role has permission');
  });

  it('checkPermission returns allowed=false when denied', async () => {
    mockFetch({ allowed: false, reason: 'no matching policy' });
    const result = await client.checkPermission('token', 'products', 'delete');
    expect(result.allowed).toBe(false);
  });

  it('assignRole sends POST to roles/{roleId}/users/{userId}', async () => {
    const fm = mockFetch(undefined);
    await client.assignRole('user-1', 'role-1');
    const [url, init] = fm.mock.calls[0];
    expect(init.method).toBe('POST');
    expect(url).toContain('/api/v1/users/user-1/roles');
  });

  it('revokeRole sends DELETE to roles/{roleId}/users/{userId}', async () => {
    const fm = mockFetch(undefined);
    await client.revokeRole('token', 'user-1', 'role-1');
    const [url, init] = fm.mock.calls[0];
    expect(init.method).toBe('DELETE');
    expect(url).toContain('/api/v1/policies/roles/role-1/users/user-1');
  });

  it('getUserRoles returns array of roles', async () => {
    mockFetch([{ id: 'r1', key: 'admin', name: 'Admin' }]);
    const result = await client.getUserRoles('token', 'user-1');
    expect(Array.isArray(result)).toBe(true);
    expect(result).toHaveLength(1);
  });

  it('listPermissions returns permission tree', async () => {
    mockFetch([{ id: 'p1', name: 'Products', resource: 'products', action: '*', children: [] }]);
    const result = await client.listPermissions('token');
    expect(Array.isArray(result)).toBe(true);
    expect(result[0].resource).toBe('products');
  });
});
