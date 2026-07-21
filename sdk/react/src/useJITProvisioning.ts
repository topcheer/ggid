/**
 * GGID React SDK — useJITProvisioning hook
 *
 * Just-In-Time provisioning configuration.
 *
 * Usage:
 *   const { config, updateConfig } = useJITProvisioning();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface JITConfig {
  enabled: boolean;
  providers: {
    id: string;
    name: string;
    type: 'saml' | 'oidc' | 'social';
    enabled: boolean;
    attribute_mapping: { claim: string; attribute: string }[];
    auto_assign_role: string | null;
    default_org: string | null;
  }[];
}

export interface UseJITProvisioningResult {
  config: JITConfig | null;
  isLoading: boolean;
  error: string | null;
  updateConfig: (input: Partial<JITConfig>) => Promise<boolean>;
  fetchConfig: () => Promise<void>;
}

export function useJITProvisioning(): UseJITProvisioningResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [config, setConfig] = useState<JITConfig | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchConfig = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/jit-provisioning`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setConfig(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const updateConfig = useCallback(async (input: Partial<JITConfig>): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/jit-provisioning`, { method: 'PUT', headers: makeHeaders(), body: JSON.stringify(input) });
      if (!resp.ok) throw new Error(`Update failed (${resp.status})`);
      setConfig(await resp.json()); return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { config, isLoading, error, updateConfig, fetchConfig };
}
