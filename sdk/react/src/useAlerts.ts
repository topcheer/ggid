/**
 * GGID React SDK — useAlerts hook
 *
 * CRUD for audit alerting rules.
 *
 * Usage:
 *   const { rules, isLoading, createRule, updateRule, deleteRule, refetch } = useAlerts();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface AlertRule {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  metric: string;
  condition: '>' | '<' | '=' | '>=' | '<=';
  threshold: number;
  window: number;
  action: 'email' | 'webhook' | 'slack' | 'pagerduty';
  target: string;
  cooldown: number;
  last_triggered?: string;
}

export interface CreateAlertRuleInput {
  name: string;
  description?: string;
  metric: string;
  condition: AlertRule['condition'];
  threshold: number;
  window: number;
  action: AlertRule['action'];
  target: string;
  cooldown?: number;
}

export interface AlertTestResult {
  matched: boolean;
  sample_count: number;
  evaluation_time_ms: number;
}

export interface UseAlertsResult {
  rules: AlertRule[];
  isLoading: boolean;
  error: string | null;
  createRule: (input: CreateAlertRuleInput) => Promise<AlertRule | null>;
  updateRule: (id: string, input: Partial<AlertRule>) => Promise<boolean>;
  deleteRule: (id: string) => Promise<boolean>;
  toggleRule: (id: string) => Promise<boolean>;
  testRule: (id: string) => Promise<AlertTestResult | null>;
  refetch: () => Promise<void>;
}

export function useAlerts(): UseAlertsResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [rules, setRules] = useState<AlertRule[]>([]);
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
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/alerting/rules`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch alert rules (${resp.status})`);
      const data = await resp.json();
      setRules(data.rules ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setRules([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchRules();
  }, [isAuthenticated, fetchRules]);

  const createRule = useCallback(
    async (input: CreateAlertRuleInput): Promise<AlertRule | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/settings/alerting/rules`, {
          method: 'POST',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to create alert rule (${resp.status})`);
        const created = await resp.json();
        await fetchRules();
        return created;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders, fetchRules]
  );

  const updateRule = useCallback(
    async (id: string, input: Partial<AlertRule>): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/settings/alerting/rules/${id}`, {
          method: 'PATCH',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to update alert rule (${resp.status})`);
        await fetchRules();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchRules]
  );

  const deleteRule = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/settings/alerting/rules/${id}`, {
          method: 'DELETE',
          headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to delete alert rule (${resp.status})`);
        await fetchRules();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchRules]
  );

  const toggleRule = useCallback(
    async (id: string): Promise<boolean> => {
      const rule = rules.find((r) => r.id === id);
      if (!rule) return false;
      return updateRule(id, { enabled: !rule.enabled });
    },
    [rules, updateRule]
  );

  const testRule = useCallback(
    async (id: string): Promise<AlertTestResult | null> => {
      try {
        const resp = await fetch(
          `${apiBaseUrl}/api/v1/settings/alerting/rules/${id}/test`,
          { method: 'POST', headers: makeHeaders() }
        );
        if (!resp.ok) throw new Error(`Failed to test alert rule (${resp.status})`);
        return await resp.json();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders]
  );

  return {
    rules,
    isLoading,
    error,
    createRule,
    updateRule,
    deleteRule,
    toggleRule,
    testRule,
    refetch: fetchRules,
  };
}
