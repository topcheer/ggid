/**
 * GGID React SDK — useRoleTemplates hook
 *
 * List, preview, and apply predefined role templates.
 *
 * Usage:
 *   const { templates, applyTemplate } = useRoleTemplates();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface PermissionNode {
  id: string;
  name: string;
  description: string;
  children: PermissionNode[];
}

export interface RoleTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  permissions: PermissionNode[];
  permission_count: number;
  system: boolean;
}

export interface ApplyResult {
  role_id: string;
  role_name: string;
  applied_permissions: number;
}

export interface UseRoleTemplatesResult {
  templates: RoleTemplate[];
  isLoading: boolean;
  error: string | null;
  applyTemplate: (templateId: string, roleName?: string) => Promise<ApplyResult | null>;
  previewTemplate: (templateId: string) => Promise<PermissionNode[] | null>;
  refetch: () => Promise<void>;
}

export function useRoleTemplates(): UseRoleTemplatesResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [templates, setTemplates] = useState<RoleTemplate[]>([]);
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

  const fetchTemplates = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/role-templates`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch templates (${resp.status})`);
      const data = await resp.json();
      setTemplates(data.templates ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setTemplates([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchTemplates();
  }, [isAuthenticated, fetchTemplates]);

  const applyTemplate = useCallback(
    async (templateId: string, roleName?: string): Promise<ApplyResult | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/policy/role-templates/${templateId}/apply`, {
          method: 'POST',
          headers: makeHeaders(),
          body: JSON.stringify({ role_name: roleName }),
        });
        if (!resp.ok) throw new Error(`Failed to apply template (${resp.status})`);
        return await resp.json();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  const previewTemplate = useCallback(
    async (templateId: string): Promise<PermissionNode[] | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/policy/role-templates/${templateId}/preview`, {
          headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to preview template (${resp.status})`);
        const data = await resp.json();
        return data.permissions ?? data.tree ?? [];
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  return {
    templates, isLoading, error,
    applyTemplate, previewTemplate,
    refetch: fetchTemplates,
  };
}
