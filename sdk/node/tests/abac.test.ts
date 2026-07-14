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

describe('ABAC evaluate', () => {
  let client: GGIDClient;

  beforeEach(() => {
    client = new GGIDClient({ gatewayUrl: 'https://iam.test.com' });
  });

  it('evaluateABAC sends POST with request body', async () => {
    const fm = mockFetch({ allowed: true, reason: 'matched rule', matched_rules: ['rule-1'] });
    const result = await client.evaluateABAC('token', {
      action: 'read',
      resource: 'document:123',
      subject: 'user:456',
      conditions: { department: 'engineering' },
    });
    expect(fm).toHaveBeenCalled();
    const [url, init] = fm.mock.calls[0];
    expect(init.method).toBe('POST');
    expect(url).toContain('/api/v1/policies/abac/evaluate');
    expect(result.allowed).toBe(true);
    expect(result.matched_rules).toContain('rule-1');
  });

  it('evaluateABAC returns allowed=false when no rules match', async () => {
    mockFetch({ allowed: false, reason: 'no matching rules', matched_rules: [] });
    const result = await client.evaluateABAC('token', {
      action: 'delete', resource: 'document:999', subject: 'user:1',
    });
    expect(result.allowed).toBe(false);
    expect(result.matched_rules).toHaveLength(0);
  });

  it('evaluateABAC with full PolicyCheckRequest', async () => {
    const fm = mockFetch({ allowed: true, reason: 'ok', matched_by: 'rbac' });
    const result = await client.evaluateABAC('token', {
      action: 'create', resource: 'orders', subject: 'user:1',
      conditions: { ip: '10.0.0.1' },
    });
    const [url, init] = fm.mock.calls[0];
    expect(init.method).toBe('POST');
    expect(url).toContain('/api/v1/policies/abac/evaluate');
    expect(result.matched_by).toBe('rbac');
  });
});
