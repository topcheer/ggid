/**
 * GGID React SDK — useRetention hook
 *
 * Fetch and update audit log retention policy.
 *
 * Usage:
 *   const { policy, isLoading, updatePolicy, refetch } = useRetention();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface RetentionPolicy {
  max_age_days: number;
  max_events: number;
  archive_enabled: boolean;
  archive_location?: string;
  delete_archived: boolean;
  compliance_mode: boolean;
}

export interface PurgeResult {
  purged_count: number;
  archived_count: number;
}

export interface RetentionSchedule {
  id: string;
  name: string;
  cron: string;
  action: 'archive' | 'purge' | 'export';
  max_age_days: number;
  enabled: boolean;
  last_run?: string;
  next_run?: string;
}

export interface CreateScheduleInput {
  name: string;
  cron: string;
  action: RetentionSchedule['action'];
  max_age_days: number;
  enabled?: boolean;
}

export interface UseRetentionResult {
  policy: RetentionPolicy | null;
  schedules: RetentionSchedule[];
  isLoading: boolean;
  error: string | null;
  updatePolicy: (policy: Partial<RetentionPolicy>) => Promise<boolean>;
  purgeOldEvents: () => Promise<PurgeResult | null>;
  createSchedule: (input: CreateScheduleInput) => Promise<RetentionSchedule | null>;
  updateSchedule: (id: string, input: Partial<RetentionSchedule>) => Promise<boolean>;
  deleteSchedule: (id: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useRetention(): UseRetentionResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [policy, setPolicy] = useState<RetentionPolicy | null>(null);
  const [schedules, setSchedules] = useState<RetentionSchedule[]>([]);
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

  const fetchPolicy = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/retention`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch retention policy (${resp.status})`);
      const data = await resp.json();
      setPolicy(data);
      setSchedules(data.schedules ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setPolicy(null);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchPolicy();
  }, [isAuthenticated, fetchPolicy]);

  const updatePolicy = useCallback(
    async (newPolicy: Partial<RetentionPolicy>): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/settings/retention`, {
          method: 'PUT',
          headers: makeHeaders(),
          body: JSON.stringify(newPolicy),
        });
        if (!resp.ok) throw new Error(`Failed to update retention policy (${resp.status})`);
        const updated = await resp.json();
        setPolicy(updated);
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders]
  );

  const purgeOldEvents = useCallback(async (): Promise<PurgeResult | null> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/retention/purge`, {
        method: 'POST', headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to purge old events (${resp.status})`);
      return await resp.json();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return null;
    }
  }, [apiBaseUrl, makeHeaders]);

  const createSchedule = useCallback(
    async (input: CreateScheduleInput): Promise<RetentionSchedule | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/settings/retention/schedules`, {
          method: 'POST', headers: makeHeaders(), body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to create schedule (${resp.status})`);
        const created = await resp.json();
        setSchedules((prev) => [...prev, created]);
        return created;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders]
  );

  const updateSchedule = useCallback(
    async (id: string, input: Partial<RetentionSchedule>): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/settings/retention/schedules/${id}`, {
          method: 'PATCH', headers: makeHeaders(), body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to update schedule (${resp.status})`);
        const updated = await resp.json();
        setSchedules((prev) => prev.map((s) => (s.id === id ? updated : s)));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders]
  );

  const deleteSchedule = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/settings/retention/schedules/${id}`, {
          method: 'DELETE', headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to delete schedule (${resp.status})`);
        setSchedules((prev) => prev.filter((s) => s.id !== id));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders]
  );

  return {
    policy,
    schedules,
    isLoading,
    error,
    updatePolicy,
    purgeOldEvents,
    createSchedule,
    updateSchedule,
    deleteSchedule,
    refetch: fetchPolicy,
  };
}
