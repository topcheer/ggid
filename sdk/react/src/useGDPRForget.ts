/**
 * GGID React SDK — useGDPRForget hook
 *
 * GDPR right-to-be-forgotten: search, execute, history.
 *
 * Usage:
 *   const { history, searchUser, execute, isLoading } = useGDPRForget();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface GDPRForgetRequest {
  id: string;
  user_id: string;
  username: string;
  email: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  requested_by: string;
  requested_at: string;
  completed_at: string;
  records_deleted: number;
  errors: string[];
}

export interface UseGDPRForgetResult {
  history: GDPRForgetRequest[];
  searchResult: { user_id: string; username: string; email: string; record_count: number } | null;
  isLoading: boolean;
  error: string | null;
  fetchHistory: () => Promise<void>;
  searchUser: (query: string) => Promise<boolean>;
  execute: (userId: string) => Promise<boolean>;
}

export function useGDPRForget(): UseGDPRForgetResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [history, setHistory] = useState<GDPRForgetRequest[]>([]);
  const [searchResult, setSearchResult] = useState<UseGDPRForgetResult['searchResult']>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchHistory = useCallback(async () => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/gdpr-forget`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setHistory(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
  }, [apiBaseUrl, makeHeaders]);

  const searchUser = useCallback(async (query: string): Promise<boolean> => {
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/gdpr-forget/search?q=${encodeURIComponent(query)}`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Search failed (${resp.status})`);
      setSearchResult(await resp.json());
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const execute = useCallback(async (userId: string): Promise<boolean> => {
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/gdpr-forget/execute`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify({ user_id: userId }) });
      if (!resp.ok) throw new Error(`Execute failed (${resp.status})`);
      await fetchHistory();
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders, fetchHistory]);

  return { history, searchResult, isLoading, error, fetchHistory, searchUser, execute };
}
