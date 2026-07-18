/**
 * GGID React SDK — useSessionRisk hook
 *
 * Session risk re-evaluation.
 *
 * Usage:
 *   const { sessions, reevaluate, isLoading } = useSessionRisk();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface SessionRiskFactor {
  type: 'ip_change' | 'device_change' | 'geo_change' | 'asn_change' | 'impossible_travel';
  detail: string;
  detected: boolean;
}

export interface SessionRiskEntry {
  session_id: string;
  user_id: string;
  username: string;
  current_risk: number;
  previous_risk: number;
  risk_delta: number;
  factors: SessionRiskFactor[];
  ip_address: string;
  device_id: string;
  location: string;
  last_evaluated: string;
  reevaluate_recommended: boolean;
}

export interface UseSessionRiskResult {
  sessions: SessionRiskEntry[];
  isLoading: boolean;
  error: string | null;
  fetchSessions: () => Promise<void>;
  reevaluate: (sessionId: string) => Promise<boolean>;
  reevaluateAll: () => Promise<boolean>;
}

export function useSessionRisk(): UseSessionRiskResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [sessions, setSessions] = useState<SessionRiskEntry[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchSessions = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/sessions/risk`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setSessions(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const reevaluate = useCallback(async (sessionId: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/sessions/${sessionId}/reevaluate`, { method: 'POST', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Re-evaluate failed (${resp.status})`);
      const updated = await resp.json() as SessionRiskEntry;
      setSessions((prev: any) => prev.map((s) => s.session_id === sessionId ? updated : s));
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const reevaluateAll = useCallback(async (): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/sessions/reevaluate-all`, { method: 'POST', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Re-evaluate all failed (${resp.status})`);
      await fetchSessions();
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders, fetchSessions]);

  return { sessions, isLoading, error, fetchSessions, reevaluate, reevaluateAll };
}
