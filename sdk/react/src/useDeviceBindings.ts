/**
 * GGID React SDK — useDeviceBindings hook
 *
 * Device binding management: list, bind, unbind.
 *
 * Usage:
 *   const { bindings, bindDevice, unbind } = useDeviceBindings();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface DeviceBinding {
  id: string;
  user_id: string;
  user_name: string;
  device_name: string;
  device_type: 'mobile' | 'desktop' | 'tablet' | 'other';
  fingerprint: string;
  bound_at: string;
  last_seen?: string;
  status: 'active' | 'revoked';
}

export interface BindDeviceInput {
  user_id: string;
  device_name: string;
  device_type?: DeviceBinding['device_type'];
  fingerprint: string;
}

export interface UseDeviceBindingsResult {
  bindings: DeviceBinding[];
  isLoading: boolean;
  error: string | null;
  bindDevice: (input: BindDeviceInput) => Promise<DeviceBinding | null>;
  unbind: (id: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useDeviceBindings(): UseDeviceBindingsResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [bindings, setBindings] = useState<DeviceBinding[]>([]);
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

  const fetchBindings = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/security/device-bindings`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch device bindings (${resp.status})`);
      const data = await resp.json();
      setBindings(data.bindings ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setBindings([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchBindings();
  }, [isAuthenticated, fetchBindings]);

  const bindDevice = useCallback(
    async (input: BindDeviceInput): Promise<DeviceBinding | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/security/device-bindings`, {
          method: 'POST', headers: makeHeaders(), body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to bind device (${resp.status})`);
        const created = await resp.json();
        setBindings((prev: any) => [...prev, created]);
        return created;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  const unbind = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/security/device-bindings/${id}`, {
          method: 'DELETE', headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to unbind device (${resp.status})`);
        setBindings((prev: any) => prev.filter((b: any) => b.id !== id));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  return {
    bindings, isLoading, error,
    bindDevice, unbind, refetch: fetchBindings,
  };
}
