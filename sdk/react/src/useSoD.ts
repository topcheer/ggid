/**
 * GGID React SDK — useSoD hook
 *
 * Separation of Duties: check violations, list/create/delete rules.
 *
 * Usage:
 *   const { rules, violations, addRule, deleteRule, checkViolation } = useSoD();
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

export interface SoDViolation {
  id: string;
  user_id: string;
  user_name: string;
  conflicting_roles: string[];
  rule_description: string;
  severity: SoDSeverity;
  detected_at: string;
}

export interface CreateSoDRuleInput {
  roles: string[];
  description: string;
  severity: SoDSeverity;
}

export interface ViolationCheckResult {
  has_violation: boolean;
  conflicting_rules: { description: string; severity: SoDSeverity }[];
}

export interface UseSoDResult {
  rules: SoDRule[];
  violations: SoDViolation[];
  isLoading: boolean;
  error: string | null;
  addRule: (input: CreateSoDRuleInput) => Promise<SoDRule | null>;
  updateRule: (id: string, input: Partial<SoDRule>) => Promise<boolean>;
  deleteRule: (id: string) => Promise<boolean>;
  checkViolation: (userId: string, roles: string[]) => Promise<ViolationCheckResult | null>;
  refetch: () => Promise<void>;
}

export function useSoD(): UseSoDResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [rules, setRules] = useState<SoDRule[]>([]);
  const [violations, setViolations] = useState<SoDViolation[]>([]);
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
      const [rulesRes, violRes] = await Promise.all([
        fetch(`${apiBaseUrl}/api/v1/policy/sod/rules`, { headers: makeHeaders() }),
        fetch(`${apiBaseUrl}/api/v1/policy/sod/violations`, { headers: makeHeaders() }),
      ]);
      let rData: { rules?: SoDRule[]; items?: SoDRule[] } = {};
      let vData: { violations?: SoDViolation[]; items?: SoDViolation[] } = {};
      if (rulesRes.ok) rData = await rulesRes.json();
      if (violRes.ok) vData = await violRes.json();
      setRules(rData.rules ?? rData.items ?? []);
      setViolations(vData.violations ?? vData.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchAll();
  }, [isAuthenticated, fetchAll]);

  const addRule = useCallback(
    async (input: CreateSoDRuleInput): Promise<SoDRule | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/policy/sod/rules`, {
          method: 'POST',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to create SoD rule (${resp.status})`);
        const created = await resp.json();
        setRules((prev: any) => [...prev, created]);
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
          method: 'PATCH',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to update SoD rule (${resp.status})`);
        const updated = await resp.json();
        setRules((prev: any) => prev.map((r) => (r.id === id ? updated : r)));
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
          method: 'DELETE',
          headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to delete SoD rule (${resp.status})`);
        setRules((prev: any) => prev.filter((r) => r.id !== id));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  const checkViolation = useCallback(
    async (userId: string, roles: string[]): Promise<ViolationCheckResult | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/policy/sod/check`, {
          method: 'POST',
          headers: makeHeaders(),
          body: JSON.stringify({ user_id: userId, roles }),
        });
        if (!resp.ok) throw new Error(`Failed to check SoD violation (${resp.status})`);
        return await resp.json();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  return {
    rules,
    violations,
    isLoading,
    error,
    addRule,
    updateRule,
    deleteRule,
    checkViolation,
    refetch: fetchAll,
  };
}
