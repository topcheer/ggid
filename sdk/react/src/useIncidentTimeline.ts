/**
 * GGID React SDK — useIncidentTimeline hook
 *
 * Incident lifecycle timeline events.
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export type IncidentPhase = 'detection' | 'triage' | 'escalation' | 'containment' | 'response' | 'resolution' | 'postmortem';

export interface IncidentTimelineEvent {
  id: string;
  incident_id: string;
  phase: IncidentPhase;
  description: string;
  actor: string;
  metadata: Record<string, string>;
  created_at: string;
}

export interface UseIncidentTimelineResult {
  events: IncidentTimelineEvent[];
  isLoading: boolean;
  error: string | null;
  fetchTimeline: (incidentId: string) => Promise<void>;
}

export function useIncidentTimeline(): UseIncidentTimelineResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [events, setEvents] = useState<IncidentTimelineEvent[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTimeline = useCallback(async (incidentId: string) => {
    const tok = getAccessToken(); if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/incidents/${incidentId}/timeline`, { headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId } });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setEvents(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [getAccessToken, apiBaseUrl, tenantId]);

  return { events, isLoading, error, fetchTimeline };
}
