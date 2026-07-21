/**
 * GGID React SDK — useProfile hook
 *
 * Current user profile get/update, change password.
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';
import type { GGIDUser } from './types';

export interface UpdateProfileInput {
  username?: string;
  email?: string;
  avatar_url?: string;
}

export interface UseProfileResult {
  profile: GGIDUser | null;
  isLoading: boolean;
  error: string | null;
  fetchProfile: () => Promise<void>;
  updateProfile: (input: UpdateProfileInput) => Promise<boolean>;
  changePassword: (oldPassword: string, newPassword: string) => Promise<boolean>;
  uploadAvatar: (file: File) => Promise<string | null>;
}

export function useProfile(): UseProfileResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [profile, setProfile] = useState<GGIDUser | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchProfile = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/users/me`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed to fetch profile (${resp.status})`);
      const data = await resp.json();
      setProfile(data.user ?? data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  const updateProfile = useCallback(async (input: UpdateProfileInput): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/users/me`, {
        method: 'PATCH', headers: makeHeaders(), body: JSON.stringify(input),
      });
      if (!resp.ok) throw new Error(`Failed to update profile (${resp.status})`);
      const data = await resp.json();
      setProfile(data.user ?? data);
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    }
  }, [apiBaseUrl, makeHeaders]);

  const changePassword = useCallback(async (oldPassword: string, newPassword: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/users/me/password`, {
        method: 'POST', headers: makeHeaders(),
        body: JSON.stringify({ old_password: oldPassword, new_password: newPassword }),
      });
      if (!resp.ok) throw new Error(`Failed to change password (${resp.status})`);
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    }
  }, [apiBaseUrl, makeHeaders]);

  const uploadAvatar = useCallback(async (file: File): Promise<string | null> => {
    try {
      const tok = getAccessToken();
      const formData = new FormData();
      formData.append('avatar', file);
      const resp = await fetch(`${apiBaseUrl}/api/v1/users/me/avatar`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId },
        body: formData,
      });
      if (!resp.ok) throw new Error(`Failed to upload avatar (${resp.status})`);
      const data = await resp.json();
      const avatarUrl = data.avatar_url ?? data.url ?? '';
      if (avatarUrl) {
        setProfile((prev) => prev ? { ...prev, avatar_url: avatarUrl } : prev);
      }
      return avatarUrl || null;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return null;
    }
  }, [getAccessToken, apiBaseUrl, tenantId]);

  return { profile, isLoading, error, fetchProfile, updateProfile, changePassword, uploadAvatar };
}
