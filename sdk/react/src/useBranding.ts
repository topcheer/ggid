/**
 * GGID React SDK — useBranding hook
 *
 * Fetch and update per-tenant branding configuration.
 *
 * Usage:
 *   const { branding, isLoading, updateBranding, refetch } = useBranding();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface BrandingConfig {
  logo_url: string;
  primary_color: string;
  secondary_color: string;
  css_override: string;
  custom_domain: string;
  email_from_name?: string;
  email_from_address?: string;
}

export interface UseBrandingResult {
  branding: BrandingConfig | null;
  isLoading: boolean;
  error: string | null;
  updateBranding: (config: Partial<BrandingConfig>) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useBranding(): UseBrandingResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [branding, setBranding] = useState<BrandingConfig | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${tok}`,
      'X-Tenant-ID': tenantId,
    };
  }, [getAccessToken, tenantId]);

  const fetchBranding = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/branding`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch branding (${resp.status})`);
      const data = await resp.json();
      setBranding(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setBranding(null);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchBranding();
  }, [isAuthenticated, fetchBranding]);

  const updateBranding = useCallback(
    async (config: Partial<BrandingConfig>): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/settings/branding`, {
          method: 'PUT',
          headers: makeHeaders(),
          body: JSON.stringify(config),
        });
        if (!resp.ok) throw new Error(`Failed to update branding (${resp.status})`);
        const updated = await resp.json();
        setBranding(updated);
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders]
  );

  return {
    branding,
    isLoading,
    error,
    updateBranding,
    refetch: fetchBranding,
  };
}
