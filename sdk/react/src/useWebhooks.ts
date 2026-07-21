/**
 * GGID React SDK — useWebhooks hook
 *
 * Webhook CRUD + event subscription + test delivery.
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface Webhook {
  id: string;
  url: string;
  description: string;
  enabled: boolean;
  events: string[];
  secret: string;
  retry_count: number;
  timeout_seconds: number;
  last_delivery?: {
    status_code: number;
    delivered_at: string;
    success: boolean;
  };
  created_at: string;
  updated_at?: string;
}

export interface CreateWebhookInput {
  url: string;
  description?: string;
  events?: string[];
  secret?: string;
  retry_count?: number;
  timeout_seconds?: number;
}

export interface UseWebhooksResult {
  webhooks: Webhook[];
  isLoading: boolean;
  error: string | null;
  createWebhook: (input: CreateWebhookInput) => Promise<Webhook | null>;
  updateWebhook: (id: string, input: Partial<Webhook>) => Promise<boolean>;
  deleteWebhook: (id: string) => Promise<boolean>;
  testWebhook: (id: string) => Promise<{ success: boolean; statusCode: number; message: string }>;
  refetch: () => Promise<void>;
}

export function useWebhooks(): UseWebhooksResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [webhooks, setWebhooks] = useState<Webhook[]>([]);
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

  const fetchWebhooks = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/webhooks`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed to fetch webhooks (${resp.status})`);
      const data = await resp.json();
      setWebhooks(data.webhooks ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setWebhooks([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchWebhooks();
  }, [isAuthenticated, fetchWebhooks]);

  const createWebhook = useCallback(async (input: CreateWebhookInput): Promise<Webhook | null> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/webhooks`, {
        method: 'POST', headers: makeHeaders(), body: JSON.stringify(input),
      });
      if (!resp.ok) throw new Error(`Failed to create webhook (${resp.status})`);
      const created = await resp.json();
      await fetchWebhooks();
      return created;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return null;
    }
  }, [apiBaseUrl, makeHeaders, fetchWebhooks]);

  const updateWebhook = useCallback(async (id: string, input: Partial<Webhook>): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/webhooks/${id}`, {
        method: 'PATCH', headers: makeHeaders(), body: JSON.stringify(input),
      });
      if (!resp.ok) throw new Error(`Failed to update webhook (${resp.status})`);
      await fetchWebhooks();
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    }
  }, [apiBaseUrl, makeHeaders, fetchWebhooks]);

  const deleteWebhook = useCallback(async (id: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/webhooks/${id}`, {
        method: 'DELETE', headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to delete webhook (${resp.status})`);
      await fetchWebhooks();
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    }
  }, [apiBaseUrl, makeHeaders, fetchWebhooks]);

  const testWebhook = useCallback(async (id: string): Promise<{ success: boolean; statusCode: number; message: string }> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/webhooks/${id}/test`, {
        method: 'POST', headers: makeHeaders(),
      });
      const data = await resp.json().catch(() => ({}));
      return {
        success: resp.ok,
        statusCode: data.status_code ?? resp.status,
        message: data.message ?? (resp.ok ? 'Delivery successful' : 'Delivery failed'),
      };
    } catch (err) {
      return { success: false, statusCode: 0, message: err instanceof Error ? err.message : 'Unknown error' };
    }
  }, [apiBaseUrl, makeHeaders]);

  return {
    webhooks, isLoading, error,
    createWebhook, updateWebhook, deleteWebhook, testWebhook,
    refetch: fetchWebhooks,
  };
}
