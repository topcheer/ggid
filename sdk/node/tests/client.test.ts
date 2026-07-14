import { describe, it, expect, vi, beforeEach } from 'vitest';
import { GGIDClient, GGIDError } from '../src/client';

describe('GGIDClient', () => {
  it('constructs with required config', () => {
    const c = new GGIDClient({ gatewayUrl: 'https://iam.test.com/' });
    expect(c).toBeInstanceOf(GGIDClient);
  });

  it('strips trailing slash from gatewayUrl', () => {
    const c = new GGIDClient({ gatewayUrl: 'https://iam.test.com/' });
    // Access private config via any cast
    const cfg = (c as any).config;
    expect(cfg.gatewayUrl).toBe('https://iam.test.com');
  });

  it('uses default tenantId when not provided', () => {
    const c = new GGIDClient({ gatewayUrl: 'https://iam.test.com' });
    const cfg = (c as any).config;
    expect(cfg.tenantId).toBe('00000000-0000-0000-0000-000000000001');
  });

  it('uses provided tenantId', () => {
    const c = new GGIDClient({ gatewayUrl: 'https://iam.test.com', tenantId: 'custom-tenant' });
    const cfg = (c as any).config;
    expect(cfg.tenantId).toBe('custom-tenant');
  });

  it('sets default timeout to 30000ms', () => {
    const c = new GGIDClient({ gatewayUrl: 'https://iam.test.com' });
    const cfg = (c as any).config;
    expect(cfg.timeout).toBe(30000);
  });

  it('respects custom timeout', () => {
    const c = new GGIDClient({ gatewayUrl: 'https://iam.test.com', timeout: 5000 });
    const cfg = (c as any).config;
    expect(cfg.timeout).toBe(5000);
  });

  it('stores apiKey when provided', () => {
    const c = new GGIDClient({ gatewayUrl: 'https://iam.test.com', apiKey: 'secret-key' });
    expect((c as any).apiKey).toBe('secret-key');
  });

  it('does not create verifier without jwksUrl', () => {
    const c = new GGIDClient({ gatewayUrl: 'https://iam.test.com' });
    expect((c as any).verifier).toBeUndefined();
  });
});

describe('GGIDError', () => {
  it('creates with statusCode and message', () => {
    const err = new GGIDError(404, 'not found');
    expect(err.statusCode).toBe(404);
    expect(err.message).toBe('not found');
    expect(err.name).toBe('GGIDError');
  });

  it('creates with code', () => {
    const err = new GGIDError(403, 'forbidden', 'forbidden');
    expect(err.code).toBe('forbidden');
  });

  it('isNotFound returns true for 404', () => {
    expect(new GGIDError(404, '').isNotFound).toBe(true);
    expect(new GGIDError(200, '').isNotFound).toBe(false);
  });

  it('isUnauthorized returns true for 401', () => {
    expect(new GGIDError(401, '').isUnauthorized).toBe(true);
    expect(new GGIDError(200, '').isUnauthorized).toBe(false);
  });

  it('isForbidden returns true for 403', () => {
    expect(new GGIDError(403, '').isForbidden).toBe(true);
    expect(new GGIDError(200, '').isForbidden).toBe(false);
  });

  it('isConflict returns true for 409', () => {
    expect(new GGIDError(409, '').isConflict).toBe(true);
    expect(new GGIDError(200, '').isConflict).toBe(false);
  });

  it('isRateLimited returns true for 429', () => {
    expect(new GGIDError(429, '').isRateLimited).toBe(true);
    expect(new GGIDError(200, '').isRateLimited).toBe(false);
  });
});
