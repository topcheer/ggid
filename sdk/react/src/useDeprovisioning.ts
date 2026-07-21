/**
 * GGID React SDK — useDeprovisioning hook
 *
 * User deprovisioning workflow: execute + history.
 *
 * Usage:
 *   const { deprovision, history, isLoading } = useDeprovisioning();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface DeprovisionResult {
  user_id: string;
  status: 'completed' | 'partial' | 'failed';
  actions: { name: string; status: 'success' | 'failed'; message: string }[];
  completed_at: string;
}

export interface DeprovisionHistoryEntry {
  id: string;
  user_id: string;
  user_name: string;
  initiated_by: string;
  status: 'completed' | 'partial' | 'failed';
  actions_summary: { total: number; success: number; failed: number };
  completed_at: string;
}

export interface DeprovisionInput {
  user_id: string;
  revoke_tokens?: boolean;
  disable_account?: boolean;
  remove_sessions?: boolean;
  transfer_data?: string;
  reason?: string;
}

export interface UseDeprovisioningResult {
  result: DeprovisionResult | null;
  history: DeprovisionHistoryEntry[];
  isLoading: boolean;
  error: string | null;
  deprovision: (input: DeprovisionInput) => Promise<DeprovisionResult | null>;
  fetchHistory: () => Promise<void>;
}

export function useDeprovisioning(): UseDeprovisioningResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [result, setResult] = useState<DeprovisionResult | null>(null);
  const [history, setHistory] = useState<DeprovisionHistoryEntry[]>([]);
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

  const deprovision = useCallback(
    async (input: DeprovisionInput): Promise<DeprovisionResult | null> => {
      setIsLoading(true);
      setError(null);
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/users/deprovision`, {
          method: 'POST', headers: makeHeaders(), body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Deprovisioning failed (${resp.status})`);
        const data = await resp.json();
        setResult(data);
        return data;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  const fetchHistory = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/users/deprovision/history?limit=20`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch history (${resp.status})`);
      const data = await resp.json();
      setHistory(data.entries ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    }
  }, [apiBaseUrl, makeHeaders]);

  return {
    result, history, isLoading, error,
    deprovision, fetchHistory,
  };
}
