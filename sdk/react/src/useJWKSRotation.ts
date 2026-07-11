/**
 * GGID React SDK — useJWKSRotation hook
 *
 * JWKS key rotation: trigger rotation, check status.
 *
 * Usage:
 *   const { status, rotate, isLoading } = useJWKSRotation();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface JWKSStatus {
  active_kid: string;
  algorithm: string;
  created_at: string;
  previous_keys: { kid: string; retired_at: string }[];
  rotation_interval_hours: number;
  next_rotation: string;
  grace_period_hours: number;
}

export interface RotationResult {
  new_kid: string;
  previous_kid: string;
  rotated_at: string;
  grace_period_ends: string;
}

export interface UseJWKSRotationResult {
  status: JWKSStatus | null;
  isLoading: boolean;
  error: string | null;
  rotate: () => Promise<RotationResult | null>;
  fetchStatus: () => Promise<void>;
}

export function useJWKSRotation(): UseJWKSRotationResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [status, setStatus] = useState<JWKSStatus | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${tok}`,
      'X-Tenant-ID': tenantId,
    };
  }, [getAccessToken, tenantId]);

  const fetchStatus = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/oauth/jwks/rotation-status`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed to fetch JWKS status (${resp.status})`);
      setStatus(await resp.json());
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setIsLoading(false);
    }
  }, [apiBaseUrl, makeHeaders]);

  const rotate = useCallback(
    async (): Promise<RotationResult | null> => {
      setIsLoading(true);
      setError(null);
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/oauth/jwks/rotate`, {
          method: 'POST', headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Rotation failed (${resp.status})`);
        const result = await resp.json();
        await fetchStatus();
        return result;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [apiBaseUrl, makeHeaders, fetchStatus],
  );

  return { status, isLoading, error, rotate, fetchStatus };
}
