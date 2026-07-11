/**
 * GGID React SDK — useTrustedDevices hook
 *
 * Trusted device management: list, toggle trust, MFA bypass.
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface TrustedDevice {
  id: string;
  user_id: string;
  username: string;
  fingerprint: string;
  platform: string;
  trusted_since: string;
  last_used: string;
  mfa_bypass_enabled: boolean;
}

export interface UseTrustedDevicesResult {
  devices: TrustedDevice[];
  isLoading: boolean;
  error: string | null;
  fetchDevices: () => Promise<void>;
  removeTrust: (id: string) => Promise<boolean>;
  toggleMfaBypass: (id: string) => Promise<boolean>;
}

export function useTrustedDevices(): UseTrustedDevicesResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [devices, setDevices] = useState<TrustedDevice[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchDevices = useCallback(async () => {
    const tok = getAccessToken(); if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/trusted-devices`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setDevices(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const removeTrust = useCallback(async (id: string): Promise<boolean> => {
    try { const resp = await fetch(`${apiBaseUrl}/api/v1/auth/trusted-devices/${id}`, { method: 'DELETE', headers: makeHeaders() }); if (!resp.ok) throw new Error(`Remove failed (${resp.status})`); setDevices((prev) => prev.filter((d) => d.id !== id)); return true; }
    catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const toggleMfaBypass = useCallback(async (id: string): Promise<boolean> => {
    const dev = devices.find((d) => d.id === id); if (!dev) return false;
    try { const resp = await fetch(`${apiBaseUrl}/api/v1/auth/trusted-devices/${id}`, { method: 'PATCH', headers: makeHeaders(), body: JSON.stringify({ mfa_bypass_enabled: !dev.mfa_bypass_enabled }) }); if (!resp.ok) throw new Error(`Toggle failed (${resp.status})`); setDevices((prev) => prev.map((d) => d.id === id ? { ...d, mfa_bypass_enabled: !d.mfa_bypass_enabled } : d)); return true; }
    catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders, devices]);

  return { devices, isLoading, error, fetchDevices, removeTrust, toggleMfaBypass };
}
