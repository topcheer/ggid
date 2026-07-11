/**
 * GGID React SDK — useDeviceTrust hook
 *
 * Device trust scoring and posture reporting.
 *
 * Usage:
 *   const { devices, config, reportPosture, updateConfig } = useDeviceTrust();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface DeviceTrustEntry {
  device_id: string;
  user_id: string;
  username: string;
  platform: string;
  os_version: string;
  managed: boolean;
  encrypted: boolean;
  jailbroken: boolean;
  last_seen: string;
  trust_score: number;
  enrolled_at: string;
}

export interface PosturePolicy {
  min_os_version: Record<string, string>;
  require_encryption: boolean;
  block_jailbreak: boolean;
  require_managed: boolean;
  min_trust_score: number;
}

export interface PostureReport {
  device_id: string;
  platform: string;
  os_version: string;
  encrypted: boolean;
  managed: boolean;
  jailbroken: boolean;
}

export interface UseDeviceTrustResult {
  devices: DeviceTrustEntry[];
  config: PosturePolicy | null;
  isLoading: boolean;
  error: string | null;
  fetchDevices: () => Promise<void>;
  fetchConfig: () => Promise<void>;
  reportPosture: (report: PostureReport) => Promise<boolean>;
  updateConfig: (patch: Partial<PosturePolicy>) => Promise<boolean>;
}

export function useDeviceTrust(): UseDeviceTrustResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [devices, setDevices] = useState<DeviceTrustEntry[]>([]);
  const [config, setConfig] = useState<PosturePolicy | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchDevices = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/devices/trust`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setDevices(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const fetchConfig = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/devices/posture/config`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setConfig(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
  }, [apiBaseUrl, makeHeaders]);

  const reportPosture = useCallback(async (report: PostureReport): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/devices/posture`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify(report) });
      if (!resp.ok) throw new Error(`Report failed (${resp.status})`);
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const updateConfig = useCallback(async (patch: Partial<PosturePolicy>): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/devices/posture/config`, { method: 'PUT', headers: makeHeaders(), body: JSON.stringify(patch) });
      if (!resp.ok) throw new Error(`Update failed (${resp.status})`);
      const updated = await resp.json() as PosturePolicy;
      setConfig(updated);
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { devices, config, isLoading, error, fetchDevices, fetchConfig, reportPosture, updateConfig };
}
