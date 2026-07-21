/**
 * GGID React SDK — useConsentScreen hook
 *
 * OAuth consent screen configuration with live preview.
 *
 * Usage:
 *   const { config, updateConfig, fetchConfig } = useConsentScreen();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface ConsentConfig {
  logo_url: string;
  primary_color: string;
  app_name: string;
  privacy_url: string;
  terms_url: string;
  support_email: string;
  custom_message: string;
  show_scopes: boolean;
  show_permissions: boolean;
  require_explicit_consent: boolean;
}

export interface UseConsentScreenResult {
  config: ConsentConfig | null;
  isLoading: boolean;
  error: string | null;
  fetchConfig: () => Promise<void>;
  updateConfig: (patch: Partial<ConsentConfig>) => Promise<boolean>;
}

export function useConsentScreen(): UseConsentScreenResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [config, setConfig] = useState<ConsentConfig | null>(null);
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
      const resp = await fetch(`${apiBaseUrl}/api/v1/oauth/consent-screen/config`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setConfig(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const updateConfig = useCallback(async (patch: Partial<ConsentConfig>): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/oauth/consent-screen/config`, { method: 'PUT', headers: makeHeaders(), body: JSON.stringify(patch) });
      if (!resp.ok) throw new Error(`Update failed (${resp.status})`);
      const updated = await resp.json() as ConsentConfig;
      setConfig(updated);
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { config, isLoading, error, fetchConfig, updateConfig };
}
