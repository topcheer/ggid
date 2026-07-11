/**
 * GGID React SDK — useWebhookDelivery hook
 *
 * Track failed webhook deliveries and retry.
 *
 * Usage:
 *   const { failed, retryDelivery } = useWebhookDelivery();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface FailedDelivery {
  id: string;
  webhook_url: string;
  event_type: string;
  attempts: number;
  last_error: string;
  last_attempt: string;
  status: 'pending_retry' | 'exhausted' | 'retrying';
  payload_preview: string;
}

export interface UseWebhookDeliveryResult {
  failed: FailedDelivery[];
  isLoading: boolean;
  error: string | null;
  retryDelivery: (id: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useWebhookDelivery(): UseWebhookDeliveryResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [failed, setFailed] = useState<FailedDelivery[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchFailed = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/webhooks/delivery-log?status=failed`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      const data = await resp.json();
      setFailed(data.entries ?? data.items ?? []);
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); setFailed([]); }
    finally { setIsLoading(false); }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => { if (isAuthenticated) fetchFailed(); }, [isAuthenticated, fetchFailed]);

  const retryDelivery = useCallback(async (id: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/webhooks/deliveries/${id}/retry`, { method: 'POST', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Retry failed (${resp.status})`);
      setFailed((prev) => prev.filter((d) => d.id !== id)); return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { failed, isLoading, error, retryDelivery, refetch: fetchFailed };
}
