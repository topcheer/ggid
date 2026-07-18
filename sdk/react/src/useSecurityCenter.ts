/**
 * GGID React SDK — useSecurityCenter hook
 *
 * Security posture score, active threats, and recommendations.
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface SecurityPosture {
  score: number;
  grade: 'A' | 'B' | 'C' | 'D' | 'F';
  total_checks: number;
  passed_checks: number;
  failed_checks: number;
  last_assessed: string;
}

export interface SecurityThreat {
  id: string;
  type: 'brute_force' | 'credential_stuffing' | 'account_takeover' | 'privilege_escalation' | 'suspicious_login';
  severity: 'low' | 'medium' | 'high' | 'critical';
  description: string;
  affected_user?: string;
  source_ip?: string;
  detected_at: string;
  status: 'active' | 'mitigated' | 'ignored';
}

export interface SecurityRecommendation {
  id: string;
  category: 'mfa' | 'password_policy' | 'access_control' | 'monitoring' | 'encryption';
  priority: 'low' | 'medium' | 'high';
  title: string;
  description: string;
  impact: string;
  remediation: string;
  status: 'open' | 'in_progress' | 'resolved';
}

export interface UseSecurityCenterResult {
  posture: SecurityPosture | null;
  threats: SecurityThreat[];
  recentThreats: SecurityThreat[];
  riskScore: number;
  riskLevel: 'low' | 'medium' | 'high' | 'critical';
  recommendations: SecurityRecommendation[];
  isLoading: boolean;
  error: string | null;
  dismissThreat: (id: string) => Promise<boolean>;
  resolveRecommendation: (id: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useSecurityCenter(): UseSecurityCenterResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [posture, setPosture] = useState<SecurityPosture | null>(null);
  const [threats, setThreats] = useState<SecurityThreat[]>([]);
  const [recommendations, setRecommendations] = useState<SecurityRecommendation[]>([]);
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

  const fetchAll = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const [postureRes, threatsRes, recsRes] = await Promise.all([
        fetch(`${apiBaseUrl}/api/v1/security/posture`, { headers: makeHeaders() }),
        fetch(`${apiBaseUrl}/api/v1/security/threats`, { headers: makeHeaders() }),
        fetch(`${apiBaseUrl}/api/v1/security/recommendations`, { headers: makeHeaders() }),
      ]);

      if (postureRes.ok) setPosture(await postureRes.json());
      if (threatsRes.ok) {
        const tData = await threatsRes.json();
        setThreats(tData.threats ?? []);
      }
      if (recsRes.ok) {
        const rData = await recsRes.json();
        setRecommendations(rData.recommendations ?? []);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchAll();
  }, [isAuthenticated, fetchAll]);

  const dismissThreat = useCallback(async (id: string): Promise<boolean> => {
    setThreats((prev: any) => prev.map((t: any) => (t.id === id ? { ...t, status: 'ignored' } : t)));
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/security/threats/${id}/dismiss`, {
        method: 'POST', headers: makeHeaders(),
      });
      return resp.ok;
    } catch {
      return false;
    }
  }, [apiBaseUrl, makeHeaders]);

  const resolveRecommendation = useCallback(async (id: string): Promise<boolean> => {
    setRecommendations((prev: any) => prev.map((r: any) => (r.id === id ? { ...r, status: 'resolved' } : r)));
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/security/recommendations/${id}/resolve`, {
        method: 'POST', headers: makeHeaders(),
      });
      return resp.ok;
    } catch {
      return false;
    }
  }, [apiBaseUrl, makeHeaders]);

  return {
    posture,
    threats,
    recentThreats: threats
      .filter((t: any) => t.status === 'active')
      .sort((a: any, b: any) => new Date(b.detected_at).getTime() - new Date(a.detected_at).getTime())
      .slice(0, 10),
    riskScore: posture ? Math.round((posture.failed_checks / Math.max(posture.total_checks, 1)) * 100) : 0,
    riskLevel: posture
      ? posture.score >= 90 ? 'low' : posture.score >= 75 ? 'medium' : posture.score >= 50 ? 'high' : 'critical'
      : 'low',
    recommendations,
    isLoading, error,
    dismissThreat, resolveRecommendation,
    refetch: fetchAll,
  };
}
