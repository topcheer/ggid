/**
 * GGID React SDK — useCorrelationRules hook
 *
 * Event correlation rules: CRUD + test.
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface CorrelationRule {
  id: string;
  name: string;
  pattern: string;
  window_minutes: number;
  threshold: number;
  enabled: boolean;
  action: string;
  created_at: string;
  last_triggered: string;
  trigger_count: number;
}

export interface UseCorrelationRulesResult {
  rules: CorrelationRule[];
  isLoading: boolean;
  error: string | null;
  fetchRules: () => Promise<void>;
  create: (rule: Omit<CorrelationRule, 'id' | 'created_at' | 'last_triggered' | 'trigger_count'>) => Promise<boolean>;
  update: (id: string, patch: Partial<CorrelationRule>) => Promise<boolean>;
  deleteRule: (id: string) => Promise<boolean>;
  test: (id: string) => Promise<{ matched: boolean; matches: number } | null>;
}

export function useCorrelationRules(): UseCorrelationRulesResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [rules, setRules] = useState<CorrelationRule[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchRules = useCallback(async () => {
    const tok = getAccessToken(); if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/correlation-rules`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setRules(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const create = useCallback(async (rule: Omit<CorrelationRule, 'id' | 'created_at' | 'last_triggered' | 'trigger_count'>) => {
    try { const resp = await fetch(`${apiBaseUrl}/api/v1/audit/correlation-rules`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify(rule) }); if (!resp.ok) throw new Error(`Create failed (${resp.status})`); await fetchRules(); return true; }
    catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders, fetchRules]);

  const update = useCallback(async (id: string, patch: Partial<CorrelationRule>) => {
    try { const resp = await fetch(`${apiBaseUrl}/api/v1/audit/correlation-rules/${id}`, { method: 'PUT', headers: makeHeaders(), body: JSON.stringify(patch) }); if (!resp.ok) throw new Error(`Update failed (${resp.status})`); setRules((prev) => prev.map((r) => r.id === id ? { ...r, ...patch } : r)); return true; }
    catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const deleteRule = useCallback(async (id: string) => {
    try { const resp = await fetch(`${apiBaseUrl}/api/v1/audit/correlation-rules/${id}`, { method: 'DELETE', headers: makeHeaders() }); if (!resp.ok) throw new Error(`Delete failed (${resp.status})`); setRules((prev) => prev.filter((r) => r.id !== id)); return true; }
    catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const test = useCallback(async (id: string) => {
    try { const resp = await fetch(`${apiBaseUrl}/api/v1/audit/correlation-rules/${id}/test`, { method: 'POST', headers: makeHeaders() }); if (!resp.ok) throw new Error(`Test failed (${resp.status})`); return await resp.json(); }
    catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return null; }
  }, [apiBaseUrl, makeHeaders]);

  return { rules, isLoading, error, fetchRules, create, update, deleteRule, test };
}
