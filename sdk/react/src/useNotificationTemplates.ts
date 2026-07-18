/**
 * GGID React SDK — useNotificationTemplates hook
 *
 * CRUD for notification templates (email/SMS/push).
 *
 * Usage:
 *   const { templates, createTemplate, updateTemplate, deleteTemplate } = useNotificationTemplates();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export type NotificationChannel = 'email' | 'sms' | 'push';

export interface NotificationTemplate {
  id: string;
  event: string;
  channel: NotificationChannel;
  subject: string;
  body: string;
  enabled: boolean;
  variables: string[];
  created_at: string;
  updated_at: string;
}

export interface CreateTemplateInput {
  event: string;
  channel: NotificationChannel;
  subject: string;
  body: string;
  variables?: string[];
}

export interface UseNotificationTemplatesResult {
  templates: NotificationTemplate[];
  isLoading: boolean;
  error: string | null;
  createTemplate: (input: CreateTemplateInput) => Promise<NotificationTemplate | null>;
  updateTemplate: (id: string, input: Partial<NotificationTemplate>) => Promise<boolean>;
  deleteTemplate: (id: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useNotificationTemplates(): UseNotificationTemplatesResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [templates, setTemplates] = useState<NotificationTemplate[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchTemplates = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/notification-templates`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      const data = await resp.json();
      setTemplates(data.templates ?? data.items ?? []);
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); setTemplates([]); }
    finally { setIsLoading(false); }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => { if (isAuthenticated) fetchTemplates(); }, [isAuthenticated, fetchTemplates]);

  const createTemplate = useCallback(async (input: CreateTemplateInput): Promise<NotificationTemplate | null> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/notification-templates`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify(input) });
      if (!resp.ok) throw new Error(`Create failed (${resp.status})`);
      const created = await resp.json(); setTemplates((prev: any) => [...prev, created]); return created;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return null; }
  }, [apiBaseUrl, makeHeaders]);

  const updateTemplate = useCallback(async (id: string, input: Partial<NotificationTemplate>): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/notification-templates/${id}`, { method: 'PATCH', headers: makeHeaders(), body: JSON.stringify(input) });
      if (!resp.ok) throw new Error(`Update failed (${resp.status})`);
      const updated = await resp.json(); setTemplates((prev: any) => prev.map((t: any) => t.id === id ? updated : t)); return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const deleteTemplate = useCallback(async (id: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/notification-templates/${id}`, { method: 'DELETE', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Delete failed (${resp.status})`);
      setTemplates((prev: any) => prev.filter((t: any) => t.id !== id)); return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { templates, isLoading, error, createTemplate, updateTemplate, deleteTemplate, refetch: fetchTemplates };
}
