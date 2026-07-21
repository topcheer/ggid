/**
 * GGID React SDK — useAuditStats hook
 *
 * Aggregate audit statistics for dashboard charts.
 *
 * Usage:
 *   const { stats, isLoading, hourlyData, topActors, refetch } = useAuditStats({ hours: 24 });
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface AuditStats {
  total_events: number;
  total_events_24h: number;
  failed_logins_24h: number;
  successful_logins_24h: number;
  unique_users_24h: number;
  unique_ips_24h: number;
  events_by_action: Record<string, number>;
  events_by_result: Record<string, number>;
}

export interface HourlyBucket {
  hour: string;
  count: number;
  failed: number;
  succeeded: number;
}

export interface TopActor {
  actor_id: string;
  actor_name: string;
  count: number;
}

export interface UseAuditStatsResult {
  stats: AuditStats | null;
  hourlyData: HourlyBucket[];
  topActors: TopActor[];
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

export function useAuditStats(options: { hours?: number } = {}): UseAuditStatsResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';
  const hours = options.hours ?? 24;

  const [stats, setStats] = useState<AuditStats | null>(null);
  const [hourlyData, setHourlyData] = useState<HourlyBucket[]>([]);
  const [topActors, setTopActors] = useState<TopActor[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(
        `${apiBaseUrl}/api/v1/audit/stats?hours=${hours}`,
        {
          headers: {
            Authorization: `Bearer ${tok}`,
            'X-Tenant-ID': tenantId,
          },
        }
      );
      if (!resp.ok) throw new Error(`Failed to fetch audit stats (${resp.status})`);
      const data = await resp.json();
      setStats({
        total_events: data.total_events ?? 0,
        total_events_24h: data.total_events_24h ?? 0,
        failed_logins_24h: data.failed_logins_24h ?? 0,
        successful_logins_24h: data.successful_logins_24h ?? 0,
        unique_users_24h: data.unique_users_24h ?? 0,
        unique_ips_24h: data.unique_ips_24h ?? 0,
        events_by_action: data.events_by_action ?? {},
        events_by_result: data.events_by_result ?? {},
      });
      setHourlyData(data.hourly_distribution ?? data.hourly ?? []);
      setTopActors(data.top_actors ?? data.topActors ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setStats(null);
      setHourlyData([]);
      setTopActors([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, tenantId, hours]);

  useEffect(() => {
    if (isAuthenticated) fetchStats();
  }, [isAuthenticated, fetchStats]);

  return {
    stats,
    hourlyData,
    topActors,
    isLoading,
    error,
    refetch: fetchStats,
  };
}
