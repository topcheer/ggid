/**
 * GGID React SDK — useComplianceSchedules hook
 *
 * CRUD for automated compliance report schedules.
 *
 * Usage:
 *   const { schedules, createSchedule, updateSchedule, deleteSchedule } = useComplianceSchedules();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export type ComplianceFramework = 'soc2' | 'hipaa' | 'gdpr' | 'iso27001' | 'pci';
export type ScheduleFrequency = 'daily' | 'weekly' | 'monthly' | 'quarterly' | 'annual';

export interface ComplianceSchedule {
  id: string;
  name: string;
  framework: ComplianceFramework;
  frequency: ScheduleFrequency;
  recipients: string[];
  next_run: string;
  last_run?: string;
  format: 'pdf' | 'csv' | 'json';
  enabled: boolean;
  created_at: string;
}

export interface CreateScheduleInput {
  name: string;
  framework: ComplianceFramework;
  frequency: ScheduleFrequency;
  recipients: string[];
  format?: 'pdf' | 'csv' | 'json';
}

export interface UseComplianceSchedulesResult {
  schedules: ComplianceSchedule[];
  isLoading: boolean;
  error: string | null;
  createSchedule: (input: CreateScheduleInput) => Promise<ComplianceSchedule | null>;
  updateSchedule: (id: string, input: Partial<ComplianceSchedule>) => Promise<boolean>;
  deleteSchedule: (id: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useComplianceSchedules(): UseComplianceSchedulesResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [schedules, setSchedules] = useState<ComplianceSchedule[]>([]);
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

  const fetchSchedules = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/compliance/schedules`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch schedules (${resp.status})`);
      const data = await resp.json();
      setSchedules(data.schedules ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setSchedules([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchSchedules();
  }, [isAuthenticated, fetchSchedules]);

  const createSchedule = useCallback(
    async (input: CreateScheduleInput): Promise<ComplianceSchedule | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/audit/compliance/schedules`, {
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
    [apiBaseUrl, makeHeaders],
  );

  const updateSchedule = useCallback(
    async (id: string, input: Partial<ComplianceSchedule>): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/audit/compliance/schedules/${id}`, {
          method: 'PATCH', headers: makeHeaders(), body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to update schedule (${resp.status})`);
        const updated = await resp.json();
        setSchedules((prev) => prev.map((s: any) => (s.id === id ? updated : s)));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  const deleteSchedule = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/audit/compliance/schedules/${id}`, {
          method: 'DELETE', headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to delete schedule (${resp.status})`);
        setSchedules((prev) => prev.filter((s: any) => s.id !== id));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  return {
    schedules, isLoading, error,
    createSchedule, updateSchedule, deleteSchedule,
    refetch: fetchSchedules,
  };
}
