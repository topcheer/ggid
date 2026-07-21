/**
 * GGID React SDK — useAuditEvents hook
 *
 * Fetches audit events with filtering and pagination.
 *
 * Usage:
 *   const { events, isLoading, error, refetch, pagination } = useAuditEvents({
 *     eventType: 'user.login',
 *     dateFrom: '2025-01-01',
 *     page: 1,
 *     pageSize: 20,
 *   });
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface AuditEvent {
  id: string;
  action: string;
  actor_id: string;
  actor_name: string;
  resource_type: string;
  resource_id: string;
  result: 'success' | 'failure' | 'denied';
  tenant_id: string;
  created_at: string;
  metadata?: Record<string, unknown>;
  ip_address?: string;
  user_agent?: string;
}

export interface AuditEventFilter {
  eventType?: string;
  resourceType?: string;
  resourceId?: string;
  actorId?: string;
  result?: string;
  dateFrom?: string;
  dateTo?: string;
  page?: number;
  pageSize?: number;
}

export interface UseAuditEventsResult {
  events: AuditEvent[];
  isLoading: boolean;
  error: string | null;
  pagination: {
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
  };
  refetch: () => Promise<void>;
}

export function useAuditEvents(filter: AuditEventFilter = {}): UseAuditEventsResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pagination, setPagination] = useState({
    page: filter.page ?? 1,
    pageSize: filter.pageSize ?? 20,
    total: 0,
    totalPages: 0,
  });

  const fetchEvents = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (filter.eventType) params.set('action', filter.eventType);
      if (filter.resourceType) params.set('resource_type', filter.resourceType);
      if (filter.resourceId) params.set('resource_id', filter.resourceId);
      if (filter.actorId) params.set('actor_id', filter.actorId);
      if (filter.result) params.set('result', filter.result);
      if (filter.dateFrom) params.set('date_from', filter.dateFrom);
      if (filter.dateTo) params.set('date_to', filter.dateTo);
      params.set('page', String(filter.page ?? 1));
      params.set('page_size', String(filter.pageSize ?? 20));

      const resp = await fetch(
        `${apiBaseUrl}/api/v1/audit/events?${params.toString()}`,
        {
          headers: {
            Authorization: `Bearer ${tok}`,
            'X-Tenant-ID': tenantId,
          },
        }
      );
      if (!resp.ok) throw new Error(`Failed to fetch audit events (${resp.status})`);
      const data = await resp.json();
      setEvents(data.events ?? data.items ?? []);
      setPagination({
        page: data.page ?? (filter.page ?? 1),
        pageSize: data.page_size ?? (filter.pageSize ?? 20),
        total: data.total ?? 0,
        totalPages: data.total_pages ?? Math.ceil((data.total ?? 0) / (filter.pageSize ?? 20)),
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setEvents([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, tenantId,
      filter.eventType, filter.resourceType, filter.resourceId,
      filter.actorId, filter.result, filter.dateFrom, filter.dateTo,
      filter.page, filter.pageSize]);

  useEffect(() => {
    if (isAuthenticated) {
      fetchEvents();
    }
  }, [isAuthenticated, fetchEvents]);

  return { events, isLoading, error, pagination, refetch: fetchEvents };
}
