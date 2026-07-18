/**
 * GGID React SDK — useIncidents hook
 *
 * Security incident management: CRUD + resolve.
 *
 * Usage:
 *   const { incidents, create, resolve, isLoading } = useIncidents();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export type IncidentSeverity = 'low' | 'medium' | 'high' | 'critical';
export type IncidentStatus = 'open' | 'investigating' | 'contained' | 'resolved' | 'closed';

export interface Incident {
  id: string;
  title: string;
  type: string;
  severity: IncidentSeverity;
  status: IncidentStatus;
  description: string;
  affected_users: string[];
  source: string;
  created_at: string;
  updated_at: string;
  resolved_at: string;
  resolution_notes: string;
  assigned_to: string;
}

export interface CreateIncidentInput {
  title: string;
  type: string;
  severity: IncidentSeverity;
  description: string;
  affected_users?: string[];
  source?: string;
}

export interface UseIncidentsResult {
  incidents: Incident[];
  isLoading: boolean;
  error: string | null;
  fetchIncidents: (status?: IncidentStatus) => Promise<void>;
  create: (input: CreateIncidentInput) => Promise<Incident | null>;
  update: (id: string, patch: Partial<Incident>) => Promise<boolean>;
  resolve: (id: string, notes: string) => Promise<boolean>;
  deleteIncident: (id: string) => Promise<boolean>;
}

export function useIncidents(): UseIncidentsResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchIncidents = useCallback(async (status?: IncidentStatus) => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const q = status ? `?status=${encodeURIComponent(status)}` : '';
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/incidents${q}`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setIncidents(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const create = useCallback(async (input: CreateIncidentInput): Promise<Incident | null> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/incidents`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify(input) });
      if (!resp.ok) throw new Error(`Create failed (${resp.status})`);
      const inc = await resp.json() as Incident;
      setIncidents((prev) => [inc, ...prev]);
      return inc;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return null; }
  }, [apiBaseUrl, makeHeaders]);

  const update = useCallback(async (id: string, patch: Partial<Incident>): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/incidents/${id}`, { method: 'PATCH', headers: makeHeaders(), body: JSON.stringify(patch) });
      if (!resp.ok) throw new Error(`Update failed (${resp.status})`);
      setIncidents((prev) => prev.map((i: any) => i.id === id ? { ...i, ...patch } : i));
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const resolve = useCallback(async (id: string, notes: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/incidents/${id}/resolve`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify({ resolution_notes: notes }) });
      if (!resp.ok) throw new Error(`Resolve failed (${resp.status})`);
      setIncidents((prev) => prev.map((i: any) => i.id === id ? { ...i, status: 'resolved', resolution_notes: notes, resolved_at: new Date().toISOString() } : i));
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const deleteIncident = useCallback(async (id: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/incidents/${id}`, { method: 'DELETE', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Delete failed (${resp.status})`);
      setIncidents((prev) => prev.filter((i: any) => i.id !== id));
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { incidents, isLoading, error, fetchIncidents, create, update, resolve, deleteIncident };
}
