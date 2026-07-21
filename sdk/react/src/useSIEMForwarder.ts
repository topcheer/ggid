/**
 * GGID React SDK — useSIEMForwarder hook
 *
 * SIEM integration: health check, test delivery, config update.
 *
 * Usage:
 *   const { status, testForwarder, updateConfig } = useSIEMForwarder();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export type SIEMFormat = 'CEF' | 'LEEF' | 'JSON' | 'syslog';

export interface SIEMConfig {
  enabled: boolean;
  destination_url: string;
  format: SIEMFormat;
  use_tls: boolean;
  batch_size: number;
  flush_interval_seconds: number;
  retry_count: number;
  filter_event_types: string[];
}

export interface SIEMStatus {
  healthy: boolean;
  last_delivery: string | null;
  total_delivered: number;
  total_failed: number;
  queue_depth: number;
  avg_latency_ms: number;
  config: SIEMConfig;
}

export interface SIEMDeliveryLog {
  id: string;
  timestamp: string;
  event_type: string;
  status: 'delivered' | 'failed' | 'retrying';
  response_code: number;
  latency_ms: number;
  error?: string;
}

export interface TestResult {
  success: boolean;
  response_time_ms: number;
  status_code: number;
  error?: string;
}

export interface UseSIEMForwarderResult {
  status: SIEMStatus | null;
  deliveryLog: SIEMDeliveryLog[];
  isLoading: boolean;
  error: string | null;
  testForwarder: () => Promise<TestResult | null>;
  updateConfig: (config: Partial<SIEMConfig>) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useSIEMForwarder(): UseSIEMForwarderResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [status, setStatus] = useState<SIEMStatus | null>(null);
  const [deliveryLog, setDeliveryLog] = useState<SIEMDeliveryLog[]>([]);
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
      const [statusRes, logRes] = await Promise.all([
        fetch(`${apiBaseUrl}/api/v1/settings/siem/status`, { headers: makeHeaders() }),
        fetch(`${apiBaseUrl}/api/v1/settings/siem/delivery-log?limit=20`, { headers: makeHeaders() }),
      ]);
      if (statusRes.ok) setStatus(await statusRes.json());
      if (logRes.ok) {
        const logData = await logRes.json();
        setDeliveryLog(logData.entries ?? logData.items ?? []);
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

  const testForwarder = useCallback(async (): Promise<TestResult | null> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/siem/test`, {
        method: 'POST', headers: makeHeaders(),
      });
      const result = await resp.json();
      return { success: resp.ok, response_time_ms: result.response_time_ms ?? 0, status_code: resp.status, error: result.error };
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return null;
    }
  }, [apiBaseUrl, makeHeaders]);

  const updateConfig = useCallback(
    async (config: Partial<SIEMConfig>): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/settings/siem/config`, {
          method: 'PUT', headers: makeHeaders(), body: JSON.stringify(config),
        });
        if (!resp.ok) throw new Error(`Failed to update SIEM config (${resp.status})`);
        await fetchAll();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchAll],
  );

  return {
    status,
    deliveryLog,
    isLoading,
    error,
    testForwarder,
    updateConfig,
    refetch: fetchAll,
  };
}
