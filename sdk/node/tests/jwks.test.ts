import { describe, it, expect, vi, beforeEach } from 'vitest';
import { JWTVerifier, JWTError } from '../src/jwt';
import '../src/introspect';

describe('JWTVerifier', () => {
  it('throws when jwksUrl not provided', () => {
    expect(() => new JWTVerifier({} as any)).toThrow('jwksUrl is required');
  });

  it('constructs with valid jwksUrl', () => {
    const v = new JWTVerifier({ jwksUrl: 'https://iam.test.com/.well-known/jwks.json' });
    expect(v).toBeInstanceOf(JWTVerifier);
  });

  it('constructs with issuer', () => {
    const v = new JWTVerifier({
      jwksUrl: 'https://iam.test.com/.well-known/jwks.json',
      issuer: 'https://iam.test.com',
    });
    expect(v).toBeInstanceOf(JWTVerifier);
  });

  it('verify throws JWTError for invalid token', async () => {
    const v = new JWTVerifier({ jwksUrl: 'https://iam.test.com/.well-known/jwks.json' });
    await expect(v.verify('invalid.token.here')).rejects.toThrow(JWTError);
  });
});

describe('Introspect', () => {
  function mockFetch(data: any) {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true, status: 200,
      json: () => Promise.resolve(data),
      text: () => Promise.resolve(JSON.stringify(data)),
    });
    (globalThis as any).fetch = fetchMock;
    return fetchMock;
  }

  it('sends POST to /api/v1/oauth/introspect', async () => {
    const { GGIDClient } = await import('../src/client');
    const client = new GGIDClient({ gatewayUrl: 'https://iam.test.com' });
    const fm = mockFetch({ active: true, sub: 'user-1', scope: 'openid profile' });
    const result = await client.introspect('admin-token', 'user-token');
    const [url, init] = fm.mock.calls[0];
    expect(init.method).toBe('POST');
    expect(url).toContain('/api/v1/oauth/introspect');
    expect(result.active).toBe(true);
    expect(result.sub).toBe('user-1');
  });

  it('returns active=false for expired token', async () => {
    const { GGIDClient } = await import('../src/client');
    const client = new GGIDClient({ gatewayUrl: 'https://iam.test.com' });
    mockFetch({ active: false });
    const result = await client.introspect('admin-token', 'expired-token');
    expect(result.active).toBe(false);
  });
});
