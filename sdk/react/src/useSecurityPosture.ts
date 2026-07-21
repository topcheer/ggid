/**
 * GGID React SDK — useSecurityPosture hook
 *
 * Overall security posture scoring and recommendations.
 *
 * Usage:
 *   const { posture, fetchPosture, isLoading } = useSecurityPosture();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface PostureRecommendation {
  id: string;
  category: string;
  title: string;
  description: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  impact: number;
  action_url: string;
}

export interface SecurityPosture {
  score: number;
  grade: 'A' | 'B' | 'C' | 'D' | 'F';
  mfa_adoption_pct: number;
  weak_password_count: number;
  total_users: number;
  active_sessions: number;
  expired_sessions: number;
  failed_logins_24h: number;
  recommendations: PostureRecommendation[];
  last_calculated: string;
}

export interface UseSecurityPostureResult {
  posture: SecurityPosture | null;
  isLoading: boolean;
  error: string | null;
  fetchPosture: () => Promise<void>;
}

export function useSecurityPosture(): UseSecurityPostureResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [posture, setPosture] = useState<SecurityPosture | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchPosture = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/security-posture`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setPosture(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  return { posture, isLoading, error, fetchPosture };
}
