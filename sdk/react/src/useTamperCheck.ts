/**
 * GGID React SDK — useTamperCheck hook
 *
 * Audit log integrity verification.
 *
 * Usage:
 *   const { status, issues, runScan, isLoading } = useTamperCheck();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface TamperIssue {
  id: string;
  event_id: string;
  type: 'hash_mismatch' | 'gap_detected' | 'chain_broken' | 'timestamp_anomaly';
  severity: 'low' | 'medium' | 'high' | 'critical';
  description: string;
  detected_at: string;
  event_timestamp: string;
}

export interface TamperStatus {
  last_scan: string;
  total_events: number;
  verified: number;
  issues_count: number;
  integrity_pct: number;
  status: 'verified' | 'warning' | 'failed';
}

export interface UseTamperCheckResult {
  status: TamperStatus | null;
  issues: TamperIssue[];
  isLoading: boolean;
  error: string | null;
  fetchStatus: () => Promise<void>;
  fetchIssues: () => Promise<void>;
  runScan: () => Promise<boolean>;
}

export function useTamperCheck(): UseTamperCheckResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [status, setStatus] = useState<TamperStatus | null>(null);
  const [issues, setIssues] = useState<TamperIssue[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchStatus = useCallback(async () => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/tamper-check/status`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setStatus(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
  }, [apiBaseUrl, makeHeaders]);

  const fetchIssues = useCallback(async () => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/tamper-check/issues`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setIssues(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
  }, [apiBaseUrl, makeHeaders]);

  const runScan = useCallback(async (): Promise<boolean> => {
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/tamper-check/scan`, { method: 'POST', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Scan failed (${resp.status})`);
      await Promise.all([fetchStatus(), fetchIssues()]);
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders, fetchStatus, fetchIssues]);

  return { status, issues, isLoading, error, fetchStatus, fetchIssues, runScan };
}
