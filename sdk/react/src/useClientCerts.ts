/**
 * GGID React SDK — useClientCerts hook
 *
 * OAuth client certificate status and rotation.
 *
 * Usage:
 *   const { certs, rotate, fetchCerts } = useClientCerts();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export type CertStatus = 'active' | 'expired' | 'revoked' | 'pending_rotation';

export interface ClientCert {
  id: string;
  client_id: string;
  client_name: string;
  cert_serial: string;
  issuer: string;
  subject: string;
  fingerprint: string;
  issued_at: string;
  expires_at: string;
  status: CertStatus;
  auto_rotate: boolean;
}

export interface RotationResult {
  cert_id: string;
  new_serial: string;
  new_fingerprint: string;
  new_expires_at: string;
}

export interface UseClientCertsResult {
  certs: ClientCert[];
  isLoading: boolean;
  error: string | null;
  fetchCerts: (clientId?: string) => Promise<void>;
  rotate: (certId: string) => Promise<RotationResult | null>;
  revoke: (certId: string) => Promise<boolean>;
  toggleAutoRotate: (certId: string) => Promise<boolean>;
}

export function useClientCerts(): UseClientCertsResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [certs, setCerts] = useState<ClientCert[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchCerts = useCallback(async (clientId?: string) => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const q = clientId ? `?client_id=${encodeURIComponent(clientId)}` : '';
      const resp = await fetch(`${apiBaseUrl}/api/v1/oauth/client-certs${q}`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setCerts(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const rotate = useCallback(async (certId: string): Promise<RotationResult | null> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/oauth/client-certs/${certId}/rotate`, { method: 'POST', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Rotate failed (${resp.status})`);
      const result = await resp.json() as RotationResult;
      await fetchCerts();
      return result;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return null; }
  }, [apiBaseUrl, makeHeaders, fetchCerts]);

  const revoke = useCallback(async (certId: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/oauth/client-certs/${certId}/revoke`, { method: 'POST', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Revoke failed (${resp.status})`);
      setCerts((prev) => prev.map((c: any) => c.id === certId ? { ...c, status: 'revoked' } : c));
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const toggleAutoRotate = useCallback(async (certId: string): Promise<boolean> => {
    try {
      const cert = certs.find((c: any) => c.id === certId);
      if (!cert) return false;
      const resp = await fetch(`${apiBaseUrl}/api/v1/oauth/client-certs/${certId}`, { method: 'PATCH', headers: makeHeaders(), body: JSON.stringify({ auto_rotate: !cert.auto_rotate }) });
      if (!resp.ok) throw new Error(`Toggle failed (${resp.status})`);
      setCerts((prev) => prev.map((c: any) => c.id === certId ? { ...c, auto_rotate: !c.auto_rotate } : c));
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders, certs]);

  return { certs, isLoading, error, fetchCerts, rotate, revoke, toggleAutoRotate };
}
