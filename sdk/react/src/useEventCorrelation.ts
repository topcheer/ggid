/**
 * GGID React SDK — useEventCorrelation hook
 *
 * Audit event correlation: rule CRUD and correlate execution.
 *
 * Usage:
 *   const { rules, correlate, createRule } = useEventCorrelation();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export type CorrelationSeverity = 'critical' | 'high' | 'medium' | 'low';

export interface CorrelationRule {
  id: string;
  name: string;
  description: string;
  event_pattern: string;
  time_window_minutes: number;
  threshold: number;
  severity: CorrelationSeverity;
  enabled: boolean;
  created_at: string;
  last_triggered?: string;
}

export interface CreateCorrelationRuleInput {
  name: string;
  description?: string;
  event_pattern: string;
  time_window_minutes: number;
  threshold: number;
  severity?: CorrelationSeverity;
}

export interface CorrelatedGroup {
  id: string;
  rule_id: string;
  rule_name: string;
  severity: CorrelationSeverity;
  event_count: number;
  events: { id: string; type: string; user: string; ip: string; timestamp: string }[];
  first_event: string;
  last_event: string;
  description: string;
}

export interface CorrelateResult {
  groups: CorrelatedGroup[];
  total_matches: number;
}

export interface UseEventCorrelationResult {
  rules: CorrelationRule[];
  results: CorrelatedGroup[];
  isLoading: boolean;
  error: string | null;
  createRule: (input: CreateCorrelationRuleInput) => Promise<CorrelationRule | null>;
  updateRule: (id: string, input: Partial<CorrelationRule>) => Promise<boolean>;
  deleteRule: (id: string) => Promise<boolean>;
  correlate: (timeWindowHours?: number) => Promise<CorrelateResult | null>;
  refetch: () => Promise<void>;
}

export function useEventCorrelation(): UseEventCorrelationResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [rules, setRules] = useState<CorrelationRule[]>([]);
  const [results, setResults] = useState<CorrelatedGroup[]>([]);
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

  const fetchRules = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/correlation/rules`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch rules (${resp.status})`);
      const data = await resp.json();
      setRules(data.rules ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchRules();
  }, [isAuthenticated, fetchRules]);

  const createRule = useCallback(
    async (input: CreateCorrelationRuleInput): Promise<CorrelationRule | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/audit/correlation/rules`, {
          method: 'POST', headers: makeHeaders(), body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to create rule (${resp.status})`);
        const created = await resp.json();
        setRules((prev) => [...prev, created]);
        return created;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  const updateRule = useCallback(
    async (id: string, input: Partial<CorrelationRule>): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/audit/correlation/rules/${id}`, {
          method: 'PATCH', headers: makeHeaders(), body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to update rule (${resp.status})`);
        const updated = await resp.json();
        setRules((prev) => prev.map((r: any) => (r.id === id ? updated : r)));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  const deleteRule = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/audit/correlation/rules/${id}`, {
          method: 'DELETE', headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to delete rule (${resp.status})`);
        setRules((prev) => prev.filter((r: any) => r.id !== id));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  const correlate = useCallback(
    async (timeWindowHours?: number): Promise<CorrelateResult | null> => {
      try {
        const params = timeWindowHours ? `?hours=${timeWindowHours}` : '';
        const resp = await fetch(`${apiBaseUrl}/api/v1/audit/correlation/analyze${params}`, {
          method: 'POST', headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to correlate events (${resp.status})`);
        const data = await resp.json();
        setResults(data.groups ?? []);
        return data;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  return {
    rules, results, isLoading, error,
    createRule, updateRule, deleteRule, correlate,
    refetch: fetchRules,
  };
}
