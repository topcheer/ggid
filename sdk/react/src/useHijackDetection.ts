/**
 * GGID React SDK — useHijackDetection hook
 *
 * Session hijack detection: suspicious sessions + terminate.
 *
 * Usage:
 *   const { suspicious, terminate, isLoading } = useHijackDetection();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface SuspiciousSession {
  session_id: string;
  user_id: string;
  username: string;
  concurrent_ips: string[];
  geo_velocity_kmh: number;
  locations: { ip: string; city: string; country: string; timestamp: string }[];
  risk_score: number;
  detected_at: string;
  reason: string;
}

export interface UseHijackDetectionResult {
  suspicious: SuspiciousSession[];
  isLoading: boolean;
  error: string | null;
  fetchSuspicious: () => Promise<void>;
  terminate: (sessionId: string) => Promise<boolean>;
}

export function useHijackDetection(): UseHijackDetectionResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [suspicious, setSuspicious] = useState<SuspiciousSession[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchSuspicious = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/sessions/hijack-detection`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setSuspicious(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const terminate = useCallback(async (sessionId: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/sessions/${sessionId}/terminate`, { method: 'POST', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Terminate failed (${resp.status})`);
      setSuspicious((prev) => prev.filter((s: any) => s.session_id !== sessionId));
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { suspicious, isLoading, error, fetchSuspicious, terminate };
}
