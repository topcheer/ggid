/**
 * GGID React SDK — useNotifications hook
 *
 * List, mark-read, and preference management.
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface Notification {
  id: string;
  type: 'info' | 'success' | 'warning' | 'error';
  title: string;
  message: string;
  read: boolean;
  created_at: string;
  action_url?: string;
  metadata?: Record<string, unknown>;
}

export interface NotificationPreferences {
  email_enabled: boolean;
  slack_enabled: boolean;
  webhook_enabled: boolean;
  categories: {
    security: boolean;
    audit: boolean;
    system: boolean;
    billing: boolean;
  };
  digest_frequency: 'realtime' | 'hourly' | 'daily' | 'weekly';
}

export interface UseNotificationsResult {
  notifications: Notification[];
  unreadCount: number;
  preferences: NotificationPreferences | null;
  isLoading: boolean;
  error: string | null;
  fetchNotifications: () => Promise<void>;
  markRead: (id: string) => Promise<boolean>;
  markAllRead: () => Promise<boolean>;
  deleteNotification: (id: string) => Promise<boolean>;
  fetchPreferences: () => Promise<void>;
  updatePreferences: (prefs: Partial<NotificationPreferences>) => Promise<boolean>;
}

export function useNotifications(): UseNotificationsResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [preferences, setPreferences] = useState<NotificationPreferences | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchNotifications = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/notifications`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed to fetch notifications (${resp.status})`);
      const data = await resp.json();
      setNotifications(data.notifications ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setNotifications([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  const markRead = useCallback(async (id: string): Promise<boolean> => {
    setNotifications((prev: any) => prev.map((n: any) => (n.id === id ? { ...n, read: true } : n)));
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/notifications/${id}/read`, { method: 'POST', headers: makeHeaders() });
      return resp.ok;
    } catch {
      return false;
    }
  }, [apiBaseUrl, makeHeaders]);

  const markAllRead = useCallback(async (): Promise<boolean> => {
    setNotifications((prev: any) => prev.map((n: any) => ({ ...n, read: true })));
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/notifications/read-all`, { method: 'POST', headers: makeHeaders() });
      return resp.ok;
    } catch {
      return false;
    }
  }, [apiBaseUrl, makeHeaders]);

  const deleteNotification = useCallback(async (id: string): Promise<boolean> => {
    setNotifications((prev: any) => prev.filter((n: any) => n.id !== id));
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/notifications/${id}`, { method: 'DELETE', headers: makeHeaders() });
      return resp.ok;
    } catch {
      return false;
    }
  }, [apiBaseUrl, makeHeaders]);

  const fetchPreferences = useCallback(async () => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/notifications/preferences`, { headers: makeHeaders() });
      if (!resp.ok) return;
      setPreferences(await resp.json());
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    }
  }, [apiBaseUrl, makeHeaders]);

  const updatePreferences = useCallback(async (prefs: Partial<NotificationPreferences>): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/notifications/preferences`, {
        method: 'PUT', headers: makeHeaders(), body: JSON.stringify(prefs),
      });
      if (!resp.ok) return false;
      const data = await resp.json();
      setPreferences(data);
      return true;
    } catch {
      return false;
    }
  }, [apiBaseUrl, makeHeaders]);

  const unreadCount = notifications.filter((n: any) => !n.read).length;

  return {
    notifications, unreadCount, preferences, isLoading, error,
    fetchNotifications, markRead, markAllRead, deleteNotification,
    fetchPreferences, updatePreferences,
  };
}
