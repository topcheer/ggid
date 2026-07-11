/**
 * GGID React SDK — useComplianceEvidence hook
 *
 * Compliance evidence collection and artifact management.
 *
 * Usage:
 *   const { evidence, uploadArtifact, exportEvidence } = useComplianceEvidence();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface EvidenceArtifact {
  id: string;
  control_id: string;
  framework: string;
  name: string;
  type: string;
  status: 'pending' | 'collected' | 'verified' | 'expired';
  collected_at: string;
  expires_at: string;
  file_url: string;
}

export interface ComplianceControl {
  id: string;
  framework: string;
  control_id: string;
  description: string;
  category: string;
  required: boolean;
  evidence_count: number;
  status: 'compliant' | 'partial' | 'missing';
}

export interface UseComplianceEvidenceResult {
  controls: ComplianceControl[];
  artifacts: EvidenceArtifact[];
  isLoading: boolean;
  error: string | null;
  fetchControls: (framework?: string) => Promise<void>;
  fetchArtifacts: (controlId?: string) => Promise<void>;
  uploadArtifact: (controlId: string, name: string, data: string) => Promise<boolean>;
  deleteArtifact: (id: string) => Promise<boolean>;
  exportEvidence: (framework: string) => Promise<boolean>;
}

export function useComplianceEvidence(): UseComplianceEvidenceResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [controls, setControls] = useState<ComplianceControl[]>([]);
  const [artifacts, setArtifacts] = useState<EvidenceArtifact[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchControls = useCallback(async (framework?: string) => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const q = framework ? `?framework=${encodeURIComponent(framework)}` : '';
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/compliance-evidence/controls${q}`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setControls(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const fetchArtifacts = useCallback(async (controlId?: string) => {
    const tok = getAccessToken();
    if (!tok) return;
    try {
      const q = controlId ? `?control_id=${encodeURIComponent(controlId)}` : '';
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/compliance-evidence/artifacts${q}`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setArtifacts(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
  }, [apiBaseUrl, makeHeaders]);

  const uploadArtifact = useCallback(async (controlId: string, name: string, data: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/compliance-evidence/artifacts`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify({ control_id: controlId, name, data }) });
      if (!resp.ok) throw new Error(`Upload failed (${resp.status})`);
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const deleteArtifact = useCallback(async (id: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/compliance-evidence/artifacts/${id}`, { method: 'DELETE', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Delete failed (${resp.status})`);
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const exportEvidence = useCallback(async (framework: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/compliance-evidence/export?framework=${encodeURIComponent(framework)}`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Export failed (${resp.status})`);
      const blob = await resp.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url; a.download = `evidence-${framework}.zip`; a.click();
      URL.revokeObjectURL(url);
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { controls, artifacts, isLoading, error, fetchControls, fetchArtifacts, uploadArtifact, deleteArtifact, exportEvidence };
}
