/**
 * GGID React SDK — useDevices hook
 *
 * WebAuthn device list + remove.
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface WebAuthnDevice {
  id: string;
  credential_id: string;
  name: string;
  device_type: 'platform' | 'cross-platform';
  authenticator_type: string;
  transports: string[];
  created_at: string;
  last_used?: string;
  aaguid?: string;
  backup_eligible?: boolean;
  backup_state?: boolean;
}

export interface UseDevicesResult {
  devices: WebAuthnDevice[];
  isLoading: boolean;
  error: string | null;
  removeDevice: (id: string) => Promise<boolean>;
  renameDevice: (id: string, name: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useDevices(): UseDevicesResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [devices, setDevices] = useState<WebAuthnDevice[]>([]);
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

  const fetchDevices = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/webauthn/credentials`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed to fetch devices (${resp.status})`);
      const data = await resp.json();
      setDevices(data.credentials ?? data.devices ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setDevices([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchDevices();
  }, [isAuthenticated, fetchDevices]);

  const removeDevice = useCallback(async (id: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/webauthn/credentials/${id}`, {
        method: 'DELETE', headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to remove device (${resp.status})`);
      setDevices((prev: any) => prev.filter((d: any) => d.id !== id));
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    }
  }, [apiBaseUrl, makeHeaders]);

  const renameDevice = useCallback(async (id: string, name: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/webauthn/credentials/${id}`, {
        method: 'PATCH', headers: makeHeaders(), body: JSON.stringify({ name }),
      });
      if (!resp.ok) throw new Error(`Failed to rename device (${resp.status})`);
      setDevices((prev: any) => prev.map((d: any) => (d.id === id ? { ...d, name } : d)));
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    }
  }, [apiBaseUrl, makeHeaders]);

  return {
    devices,
    isLoading,
    error,
    removeDevice,
    renameDevice,
    refetch: fetchDevices,
  };
}
