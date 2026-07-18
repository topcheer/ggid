/**
 * GGID React SDK — useAnomalies hook
 *
 * Anomaly detection: list, dismiss, escalate.
 *
 * Usage:
 *   const { anomalies, dismiss, escalate, isLoading } = useAnomalies();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export type AnomalySeverity = 'low' | 'medium' | 'high' | 'critical';
export type AnomalyStatus = 'active' | 'dismissed' | 'escalated' | 'resolved';

export interface Anomaly {
  id: string;
  type: string;
  description: string;
  severity: AnomalySeverity;
  confidence: number;
  status: AnomalyStatus;
  user_id: string;
  related_events: { event_id: string; action: string; timestamp: string }[];
  detected_at: string;
  metadata: Record<string, string>;
}

export interface UseAnomaliesResult {
  anomalies: Anomaly[];
  isLoading: boolean;
  error: string | null;
  fetchAnomalies: (status?: AnomalyStatus) => Promise<void>;
  dismiss: (id: string, reason: string) => Promise<boolean>;
  escalate: (id: string, note: string) => Promise<boolean>;
}

export function useAnomalies(): UseAnomaliesResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [anomalies, setAnomalies] = useState<Anomaly[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchAnomalies = useCallback(async (status?: AnomalyStatus) => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const q = status ? `?status=${encodeURIComponent(status)}` : '';
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/anomalies${q}`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setAnomalies(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const dismiss = useCallback(async (id: string, reason: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/anomalies/${id}/dismiss`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify({ reason }) });
      if (!resp.ok) throw new Error(`Dismiss failed (${resp.status})`);
      setAnomalies((prev: any) => prev.map((a) => a.id === id ? { ...a, status: 'dismissed' } : a));
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const escalate = useCallback(async (id: string, note: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/anomalies/${id}/escalate`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify({ note }) });
      if (!resp.ok) throw new Error(`Escalate failed (${resp.status})`);
      setAnomalies((prev: any) => prev.map((a) => a.id === id ? { ...a, status: 'escalated' } : a));
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { anomalies, isLoading, error, fetchAnomalies, dismiss, escalate };
}
