/**
 * GGID React SDK — useComplianceMapping hook
 *
 * Compliance framework control mapping.
 *
 * Usage:
 *   const { mappings, fetchMappings, updateMapping } = useComplianceMapping();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export type MappingStatus = 'compliant' | 'partial' | 'missing' | 'not_applicable';

export interface ControlMapping {
  id: string;
  framework: string;
  control_id: string;
  requirement: string;
  category: string;
  status: MappingStatus;
  evidence_count: number;
  mapped_policies: string[];
  gaps: string[];
}

export interface UseComplianceMappingResult {
  mappings: ControlMapping[];
  isLoading: boolean;
  error: string | null;
  fetchMappings: (framework?: string) => Promise<void>;
  updateMapping: (id: string, patch: Partial<ControlMapping>) => Promise<boolean>;
}

export function useComplianceMapping(): UseComplianceMappingResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [mappings, setMappings] = useState<ControlMapping[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchMappings = useCallback(async (framework?: string) => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const q = framework ? `?framework=${encodeURIComponent(framework)}` : '';
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/compliance-mapping${q}`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setMappings(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const updateMapping = useCallback(async (id: string, patch: Partial<ControlMapping>): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/compliance-mapping/${id}`, { method: 'PUT', headers: makeHeaders(), body: JSON.stringify(patch) });
      if (!resp.ok) throw new Error(`Update failed (${resp.status})`);
      setMappings((prev: any) => prev.map((m: any) => m.id === id ? { ...m, ...patch } : m));
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { mappings, isLoading, error, fetchMappings, updateMapping };
}
