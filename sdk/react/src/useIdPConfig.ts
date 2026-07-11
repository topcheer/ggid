/**
 * GGID React SDK — useIdPConfig hook
 *
 * IdP config CRUD (SAML/OIDC/LDAP) + test connection.
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface IdPConfig {
  id: string;
  name: string;
  type: 'saml' | 'oidc' | 'ldap';
  enabled: boolean;
  domain?: string;
  autoProvision?: boolean;
  // SAML
  entityId?: string;
  ssoUrl?: string;
  sloUrl?: string;
  certFingerprint?: string;
  // OIDC
  issuerUrl?: string;
  clientId?: string;
  clientSecret?: string;
  scopes?: string;
  authorizationEndpoint?: string;
  tokenEndpoint?: string;
  userinfoEndpoint?: string;
  // LDAP
  ldapUrl?: string;
  bindDn?: string;
  baseDn?: string;
  userFilter?: string;
  startTls?: boolean;
  // Attribute mapping
  attributeMapping?: Record<string, string>;
}

export interface CreateIdPInput {
  name: string;
  type: 'saml' | 'oidc' | 'ldap';
  domain?: string;
  autoProvision?: boolean;
}

export interface UseIdPConfigResult {
  configs: IdPConfig[];
  isLoading: boolean;
  error: string | null;
  createConfig: (input: CreateIdPInput & Partial<IdPConfig>) => Promise<IdPConfig | null>;
  updateConfig: (id: string, input: Partial<IdPConfig>) => Promise<boolean>;
  deleteConfig: (id: string) => Promise<boolean>;
  testConnection: (id: string) => Promise<{ success: boolean; message: string }>;
  refetch: () => Promise<void>;
}

export function useIdPConfig(): UseIdPConfigResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [configs, setConfigs] = useState<IdPConfig[]>([]);
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

  const fetchConfigs = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/idp`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed to fetch IdP configs (${resp.status})`);
      const data = await resp.json();
      setConfigs(data.configs ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setConfigs([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchConfigs();
  }, [isAuthenticated, fetchConfigs]);

  const createConfig = useCallback(async (input: CreateIdPInput & Partial<IdPConfig>): Promise<IdPConfig | null> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/idp`, {
        method: 'POST', headers: makeHeaders(), body: JSON.stringify(input),
      });
      if (!resp.ok) throw new Error(`Failed to create IdP config (${resp.status})`);
      const created = await resp.json();
      await fetchConfigs();
      return created;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return null;
    }
  }, [apiBaseUrl, makeHeaders, fetchConfigs]);

  const updateConfig = useCallback(async (id: string, input: Partial<IdPConfig>): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/idp/${id}`, {
        method: 'PATCH', headers: makeHeaders(), body: JSON.stringify(input),
      });
      if (!resp.ok) throw new Error(`Failed to update IdP config (${resp.status})`);
      await fetchConfigs();
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    }
  }, [apiBaseUrl, makeHeaders, fetchConfigs]);

  const deleteConfig = useCallback(async (id: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/idp/${id}`, {
        method: 'DELETE', headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to delete IdP config (${resp.status})`);
      await fetchConfigs();
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    }
  }, [apiBaseUrl, makeHeaders, fetchConfigs]);

  const testConnection = useCallback(async (id: string): Promise<{ success: boolean; message: string }> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/idp/${id}/test`, {
        method: 'POST', headers: makeHeaders(),
      });
      const data = await resp.json();
      return { success: resp.ok, message: data.message ?? (resp.ok ? 'Connection successful' : 'Connection failed') };
    } catch (err) {
      return { success: false, message: err instanceof Error ? err.message : 'Unknown error' };
    }
  }, [apiBaseUrl, makeHeaders]);

  const uploadSAMLMetadata = useCallback(async (id: string, xml: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/idp/${id}/saml-metadata`, {
        method: 'POST',
        headers: { ...makeHeaders(), 'Content-Type': 'application/xml' },
        body: xml,
      });
      if (!resp.ok) throw new Error(`Metadata upload failed (${resp.status})`);
      await fetchConfigs();
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    }
  }, [apiBaseUrl, makeHeaders, fetchConfigs]);

  const downloadSPMetadata = useCallback(async (id: string): Promise<string | null> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/idp/${id}/sp-metadata`, {
        headers: { ...makeHeaders(), Accept: 'application/xml' },
      });
      if (!resp.ok) throw new Error(`Metadata download failed (${resp.status})`);
      return await resp.text();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return null;
    }
  }, [apiBaseUrl, makeHeaders]);

  return {
    configs, isLoading, error,
    createConfig, updateConfig, deleteConfig,
    testConnection, uploadSAMLMetadata, downloadSPMetadata,
    refetch: fetchConfigs,
  };
}
