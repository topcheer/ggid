/**
 * GGID React SDK — useSoDRules hook
 *
 * Separation of Duties rule management.
 *
 * Usage:
 *   const { rules, createRule, updateRule, deleteRule } = useSoDRules();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export type SoDSeverity = 'critical' | 'high' | 'medium';

export interface SoDRule {
  id: string;
  roles: string[];
  description: string;
  severity: SoDSeverity;
  enabled: boolean;
  created_at: string;
}

export interface CreateSoDRuleInput {
  roles: string[];
  description: string;
  severity?: SoDSeverity;
}

export interface UseSoDRulesResult {
  rules: SoDRule[];
  isLoading: boolean;
  error: string | null;
  createRule: (input: CreateSoDRuleInput) => Promise<SoDRule | null>;
  updateRule: (id: string, input: Partial<SoDRule>) => Promise<boolean>;
  deleteRule: (id: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useSoDRules(): UseSoDRulesResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [rules, setRules] = useState<SoDRule[]>([]);
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
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/sod/rules`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed to fetch SoD rules (${resp.status})`);
      const data = await resp.json();
      setRules(data.rules ?? data.items ?? []);
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
    async (input: CreateSoDRuleInput): Promise<SoDRule | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/policy/sod/rules`, {
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
    async (id: string, input: Partial<SoDRule>): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/policy/sod/rules/${id}`, {
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
        const resp = await fetch(`${apiBaseUrl}/api/v1/policy/sod/rules/${id}`, {
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

  return {
    rules, isLoading, error,
    createRule, updateRule, deleteRule,
    refetch: fetchRules,
  };
}
