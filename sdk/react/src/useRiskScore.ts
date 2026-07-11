/**
 * GGID React SDK — useRiskScore hook
 *
 * User risk scoring and factor analysis.
 *
 * Usage:
 *   const { scores, highRisk, recalculate, isLoading } = useRiskScore();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface RiskFactor {
  name: string;
  weight: number;
  value: number;
  description: string;
}

export interface UserRiskScore {
  user_id: string;
  username: string;
  email: string;
  score: number;
  level: 'low' | 'medium' | 'high' | 'critical';
  factors: RiskFactor[];
  last_updated: string;
}

export interface RiskScoreSummary {
  total_users: number;
  average_score: number;
  high_risk_count: number;
  critical_count: number;
  top_factors: RiskFactor[];
  distribution: { level: string; count: number }[];
}

export interface UseRiskScoreResult {
  scores: UserRiskScore[];
  summary: RiskScoreSummary | null;
  isLoading: boolean;
  error: string | null;
  recalculate: (userId: string) => Promise<boolean>;
  fetchScores: () => Promise<void>;
  fetchSummary: () => Promise<void>;
}

export function useRiskScore(): UseRiskScoreResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [scores, setScores] = useState<UserRiskScore[]>([]);
  const [summary, setSummary] = useState<RiskScoreSummary | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchScores = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/risk-score/users`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setScores(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const fetchSummary = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/risk-score/summary`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setSummary(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
  }, [apiBaseUrl, makeHeaders]);

  const recalculate = useCallback(async (userId: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/risk-score/recalculate`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify({ user_id: userId }) });
      if (!resp.ok) throw new Error(`Recalculate failed (${resp.status})`);
      await fetchScores(); return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders, fetchScores]);

  return { scores, summary, isLoading, error, recalculate, fetchScores, fetchSummary };
}
