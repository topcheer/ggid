/**
 * GGID React SDK — useUserTimeline hook
 *
 * User activity timeline: logins, MFA, role changes, etc.
 *
 * Usage:
 *   const { events, fetchTimeline, isLoading } = useUserTimeline();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export type TimelineEventType = 'login' | 'logout' | 'password_change' | 'mfa_enroll' | 'mfa_remove' | 'role_change' | 'permission_grant' | 'permission_revoke' | 'session_revoke' | 'api_key_create' | 'api_key_revoke' | 'profile_update';

export interface TimelineEvent {
  id: string;
  event_type: TimelineEventType;
  description: string;
  actor: string;
  ip_address: string;
  user_agent: string;
  metadata: Record<string, string>;
  created_at: string;
}

export interface UseUserTimelineResult {
  events: TimelineEvent[];
  isLoading: boolean;
  error: string | null;
  fetchTimeline: (userId: string, eventType?: TimelineEventType) => Promise<void>;
}

export function useUserTimeline(): UseUserTimelineResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [events, setEvents] = useState<TimelineEvent[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTimeline = useCallback(async (userId: string, eventType?: TimelineEventType) => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const params = new URLSearchParams();
      if (eventType) params.set('type', eventType);
      const q = params.toString() ? `&${params.toString()}` : '';
      const resp = await fetch(`${apiBaseUrl}/api/v1/users/${userId}/timeline?tenant_id=${encodeURIComponent(tenantId)}${q}`, { headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId } });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setEvents(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [getAccessToken, apiBaseUrl, tenantId]);

  return { events, isLoading, error, fetchTimeline };
}
