import { describe, it, expect, vi, beforeEach } from 'vitest';
import { GGIDClient, GGIDError } from '../src/client';

describe('Auth methods', () => {
  let client: GGIDClient;
  let mockRequest: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    client = new GGIDClient({ gatewayUrl: 'https://iam.test.com' });
    mockRequest = vi.spyOn(client as any, 'request');
  });

  it('login sends POST to /api/v1/auth/login', async () => {
    const tokenSet = { access_token: 'tok', token_type: 'Bearer', expires_in: 3600 };
    mockRequest.mockResolvedValue(tokenSet);

    const result = await client.login({ username: 'admin', password: 'pass' });
    expect(mockRequest).toHaveBeenCalledWith('POST', '/api/v1/auth/login', { username: 'admin', password: 'pass' });
    expect(result.access_token).toBe('tok');
    expect(result.expires_in).toBe(3600);
  });

  it('register sends POST with username, email, password', async () => {
    mockRequest.mockResolvedValue({ user_id: 'u1' });
    const result = await client.register('user1', 'user1@test.com', 'pass');
    expect(mockRequest).toHaveBeenCalledWith('POST', '/api/v1/auth/register', {
      username: 'user1', email: 'user1@test.com', password: 'pass',
    });
    expect(result.user_id).toBe('u1');
  });

  it('logout sends POST with access_token', async () => {
    mockRequest.mockResolvedValue(undefined);
    await client.logout('my-token');
    expect(mockRequest).toHaveBeenCalledWith('POST', '/api/v1/auth/logout', { access_token: 'my-token' });
  });

  it('refreshToken sends POST with refresh_token', async () => {
    const newTokens = { access_token: 'new-tok', token_type: 'Bearer', expires_in: 3600 };
    mockRequest.mockResolvedValue(newTokens);
    const result = await client.refreshToken('refresh-tok');
    expect(mockRequest).toHaveBeenCalledWith('POST', '/api/v1/auth/refresh', { refresh_token: 'refresh-tok' });
    expect(result.access_token).toBe('new-tok');
  });

  it('verifyToken throws when no verifier configured', async () => {
    await expect(client.verifyToken('token')).rejects.toThrow('no jwksUrl configured');
  });
});

describe('TokenManager', () => {
  it('can be constructed with a client', async () => {
    const { TokenManager } = await import('../src/token_manager');
    const c = new GGIDClient({ gatewayUrl: 'https://iam.test.com' });
    const tm = new TokenManager(c, { username: 'admin', password: 'pass' });
    expect(tm).toBeDefined();
  });
});
