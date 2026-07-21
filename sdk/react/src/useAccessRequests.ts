/**
 * GGID React SDK — useAccessRequests hook
 *
 * IGA access request workflow: list, create, approve, reject.
 *
 * Usage:
 *   const { requests, isLoading, createRequest, approveRequest, rejectRequest } = useAccessRequests();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface AccessRequest {
  id: string;
  requester_id: string;
  requester_name: string;
  resource_type: string;
  resource_id: string;
  requested_role: string;
  justification: string;
  status: 'pending' | 'approved' | 'rejected' | 'expired' | 'revoked';
  created_at: string;
  updated_at: string;
  reviewer_id?: string;
  reviewer_name?: string;
  review_comment?: string;
  expires_at?: string;
}

export interface CreateAccessRequestInput {
  resource_type: string;
  resource_id: string;
  requested_role: string;
  justification: string;
  duration_days?: number;
}

export interface UseAccessRequestsResult {
  requests: AccessRequest[];
  isLoading: boolean;
  error: string | null;
  createRequest: (input: CreateAccessRequestInput) => Promise<AccessRequest | null>;
  approveRequest: (id: string, comment?: string) => Promise<boolean>;
  rejectRequest: (id: string, comment?: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useAccessRequests(statusFilter?: string): UseAccessRequestsResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [requests, setRequests] = useState<AccessRequest[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchRequests = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (statusFilter) params.set('status', statusFilter);
      const resp = await fetch(
        `${apiBaseUrl}/api/v1/access-requests?${params.toString()}`,
        {
          headers: {
            Authorization: `Bearer ${tok}`,
            'X-Tenant-ID': tenantId,
          },
        }
      );
      if (!resp.ok) throw new Error(`Failed to fetch access requests (${resp.status})`);
      const data = await resp.json();
      setRequests(data.requests ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setRequests([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, tenantId, statusFilter]);

  useEffect(() => {
    if (isAuthenticated) {
      fetchRequests();
    }
  }, [isAuthenticated, fetchRequests]);

  const createRequest = useCallback(
    async (input: CreateAccessRequestInput): Promise<AccessRequest | null> => {
      const tok = getAccessToken();
      if (!tok) return null;
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/access-requests`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${tok}`,
            'X-Tenant-ID': tenantId,
          },
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to create access request (${resp.status})`);
        const created = await resp.json();
        await fetchRequests();
        return created;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [getAccessToken, apiBaseUrl, tenantId, fetchRequests]
  );

  const approveRequest = useCallback(
    async (id: string, comment?: string): Promise<boolean> => {
      const tok = getAccessToken();
      if (!tok) return false;
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/access-requests/${id}/approve`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${tok}`,
            'X-Tenant-ID': tenantId,
          },
          body: JSON.stringify({ comment }),
        });
        if (!resp.ok) throw new Error(`Failed to approve (${resp.status})`);
        await fetchRequests();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [getAccessToken, apiBaseUrl, tenantId, fetchRequests]
  );

  const rejectRequest = useCallback(
    async (id: string, comment?: string): Promise<boolean> => {
      const tok = getAccessToken();
      if (!tok) return false;
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/access-requests/${id}/reject`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${tok}`,
            'X-Tenant-ID': tenantId,
          },
          body: JSON.stringify({ comment }),
        });
        if (!resp.ok) throw new Error(`Failed to reject (${resp.status})`);
        await fetchRequests();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [getAccessToken, apiBaseUrl, tenantId, fetchRequests]
  );

  return {
    requests,
    isLoading,
    error,
    createRequest,
    approveRequest,
    rejectRequest,
    refetch: fetchRequests,
  };
}
