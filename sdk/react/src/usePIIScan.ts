/**
 * GGID React SDK — usePIIScan hook
 *
 * PII discovery scanning and results management.
 *
 * Usage:
 *   const { results, runScan, isLoading } = usePIIScan();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface PIIFinding {
  id: string;
  field: string;
  entity: string;
  entity_type: string;
  pii_type: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  count: number;
  sample: string;
  discovered_at: string;
}

export interface PIIScanResult {
  scan_id: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  started_at: string;
  completed_at: string;
  total_findings: number;
  critical: number;
  high: number;
  medium: number;
  low: number;
  findings: PIIFinding[];
}

export interface PIIScanSummary {
  last_scan: string;
  total_scans: number;
  total_findings: number;
  unresolved: number;
  pii_types: { type: string; count: number }[];
}

export interface UsePIIScanResult {
  results: PIIScanResult | null;
  summary: PIIScanSummary | null;
  isLoading: boolean;
  error: string | null;
  runScan: () => Promise<boolean>;
  fetchResults: () => Promise<void>;
  fetchSummary: () => Promise<void>;
}

export function usePIIScan(): UsePIIScanResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [results, setResults] = useState<PIIScanResult | null>(null);
  const [summary, setSummary] = useState<PIIScanSummary | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchResults = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/pii-scan/results`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setResults(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const fetchSummary = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/pii-scan/summary`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setSummary(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
  }, [apiBaseUrl, makeHeaders]);

  const runScan = useCallback(async (): Promise<boolean> => {
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/pii-scan/run`, { method: 'POST', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Scan failed (${resp.status})`);
      await fetchResults(); return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders, fetchResults]);

  return { results, summary, isLoading, error, runScan, fetchResults, fetchSummary };
}
