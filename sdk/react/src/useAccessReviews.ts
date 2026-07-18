/**
 * GGID React SDK — useAccessReviews hook
 *
 * Access recertification campaign management.
 *
 * Usage:
 *   const { campaigns, createCampaign, submitReview } = useAccessReviews();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface AccessReviewCampaign {
  id: string;
  name: string;
  scope: string;
  reviewer: string;
  deadline: string;
  status: 'pending' | 'in_progress' | 'completed' | 'overdue';
  total_items: number;
  reviewed_items: number;
  created_at: string;
}

export interface ReviewItem {
  id: string;
  user_id: string;
  user_name: string;
  role: string;
  last_accessed: string;
  decision?: 'approve' | 'revoke';
}

export interface CreateCampaignInput {
  name: string;
  scope: string;
  reviewer: string;
  deadline: string;
}

export interface UseAccessReviewsResult {
  campaigns: AccessReviewCampaign[];
  items: ReviewItem[];
  isLoading: boolean;
  error: string | null;
  createCampaign: (input: CreateCampaignInput) => Promise<AccessReviewCampaign | null>;
  submitReview: (campaignId: string, itemId: string, decision: 'approve' | 'revoke') => Promise<boolean>;
  deleteCampaign: (id: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useAccessReviews(): UseAccessReviewsResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [campaigns, setCampaigns] = useState<AccessReviewCampaign[]>([]);
  const [items, setItems] = useState<ReviewItem[]>([]);
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

  const fetchCampaigns = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/access-reviews`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed to fetch campaigns (${resp.status})`);
      const data = await resp.json();
      setCampaigns(data.campaigns ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setCampaigns([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchCampaigns();
  }, [isAuthenticated, fetchCampaigns]);

  const createCampaign = useCallback(
    async (input: CreateCampaignInput): Promise<AccessReviewCampaign | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/audit/access-reviews`, {
          method: 'POST', headers: makeHeaders(), body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to create campaign (${resp.status})`);
        const created = await resp.json();
        await fetchCampaigns();
        return created;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders, fetchCampaigns],
  );

  const submitReview = useCallback(
    async (campaignId: string, itemId: string, decision: 'approve' | 'revoke'): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/audit/access-reviews/${campaignId}/items/${itemId}`, {
          method: 'POST', headers: makeHeaders(), body: JSON.stringify({ decision }),
        });
        if (!resp.ok) throw new Error(`Failed to submit review (${resp.status})`);
        setItems((prev) => prev.filter((i: any) => i.id !== itemId));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  const deleteCampaign = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/audit/access-reviews/${id}`, {
          method: 'DELETE', headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to delete campaign (${resp.status})`);
        setCampaigns((prev) => prev.filter((c: any) => c.id !== id));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  return {
    campaigns, items, isLoading, error,
    createCampaign, submitReview, deleteCampaign,
    refetch: fetchCampaigns,
  };
}
