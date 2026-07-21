/**
 * GGID React SDK — useApprovals hook
 *
 * Approval workflow: pending list, approve/reject.
 *
 * Usage:
 *   const { pending, approve, reject, isLoading } = useApprovals();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export type ApprovalStatus = 'pending' | 'approved' | 'rejected' | 'expired' | 'cancelled';

export interface ApprovalStep {
  step: number;
  name: string;
  approver: string;
  status: 'pending' | 'approved' | 'rejected' | 'skipped';
  acted_at: string;
  comment: string;
}

export interface ApprovalRequest {
  id: string;
  request_type: string;
  requester: string;
  requester_name: string;
  description: string;
  current_step: number;
  total_steps: number;
  approver_chain: ApprovalStep[];
  status: ApprovalStatus;
  created_at: string;
  expires_at: string;
}

export interface UseApprovalsResult {
  pending: ApprovalRequest[];
  history: ApprovalRequest[];
  isLoading: boolean;
  error: string | null;
  fetchPending: () => Promise<void>;
  fetchHistory: () => Promise<void>;
  approve: (id: string, comment: string) => Promise<boolean>;
  reject: (id: string, comment: string) => Promise<boolean>;
  cancel: (id: string) => Promise<boolean>;
}

export function useApprovals(): UseApprovalsResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [pending, setPending] = useState<ApprovalRequest[]>([]);
  const [history, setHistory] = useState<ApprovalRequest[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchPending = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/approvals?status=pending`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setPending(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const fetchHistory = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/approvals?status=completed`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setHistory(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
  }, [apiBaseUrl, makeHeaders]);

  const approve = useCallback(async (id: string, comment: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/approvals/${id}/approve`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify({ comment }) });
      if (!resp.ok) throw new Error(`Approve failed (${resp.status})`);
      await fetchPending();
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders, fetchPending]);

  const reject = useCallback(async (id: string, comment: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/approvals/${id}/reject`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify({ comment }) });
      if (!resp.ok) throw new Error(`Reject failed (${resp.status})`);
      await fetchPending();
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders, fetchPending]);

  const cancel = useCallback(async (id: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/approvals/${id}/cancel`, { method: 'POST', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Cancel failed (${resp.status})`);
      await fetchPending();
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders, fetchPending]);

  return { pending, history, isLoading, error, fetchPending, fetchHistory, approve, reject, cancel };
}
