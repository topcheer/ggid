/**
 * GGID React SDK — useMFA hook
 *
 * TOTP enroll/verify, WebAuthn register, backup codes.
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface MFAStatus {
  totp_enabled: boolean;
  webauthn_enabled: boolean;
  backup_codes_remaining: number;
}

export interface TOTPSecret {
  secret: string;
  qr_code_url: string;
}

export interface BackupCodes {
  codes: string[];
  generated_at: string;
}

export interface WebAuthnCredential {
  id: string;
  name: string;
  device_type: string;
  platform: 'platform' | 'cross-platform';
  authenticator_type?: string;
  aaguid?: string;
  backup_eligible?: boolean;
  backup_state?: boolean;
  created_at: string;
  last_used?: string;
}

export interface UseMFAResult {
  status: MFAStatus | null;
  credentials: WebAuthnCredential[];
  isLoading: boolean;
  error: string | null;
  enrollTOTP: () => Promise<TOTPSecret | null>;
  verifyTOTP: (code: string) => Promise<boolean>;
  disableTOTP: () => Promise<boolean>;
  generateBackupCodes: () => Promise<BackupCodes | null>;
  registerWebAuthn: (name: string) => Promise<boolean>;
  registerPlatformAuthenticator: (name: string) => Promise<boolean>;
  unregisterWebAuthn: (id: string) => Promise<boolean>;
  isPlatformSupported: boolean;
  platformCredentials: WebAuthnCredential[];
  crossPlatformCredentials: WebAuthnCredential[];
  fetchStatus: () => Promise<void>;
}

export function useMFA(): UseMFAResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [status, setStatus] = useState<MFAStatus | null>(null);
  const [credentials, setCredentials] = useState<WebAuthnCredential[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isPlatformSupported, setIsPlatformSupported] = useState(false);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchStatus = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    try {
      const [sRes, cRes] = await Promise.all([
        fetch(`${apiBaseUrl}/api/v1/mfa/status`, { headers: makeHeaders() }),
        fetch(`${apiBaseUrl}/api/v1/webauthn/credentials`, { headers: makeHeaders() }),
      ]);
      if (sRes.ok) setStatus(await sRes.json());
      if (cRes.ok) {
        const cData = await cRes.json();
        setCredentials(cData.credentials ?? cData.items ?? []);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  const enrollTOTP = useCallback(async (): Promise<TOTPSecret | null> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/mfa/totp/enroll`, { method: 'POST', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Enroll failed (${resp.status})`);
      return await resp.json();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return null;
    }
  }, [apiBaseUrl, makeHeaders]);

  const verifyTOTP = useCallback(async (code: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/mfa/totp/verify`, {
        method: 'POST', headers: makeHeaders(), body: JSON.stringify({ code }),
      });
      if (!resp.ok) return false;
      await fetchStatus();
      return true;
    } catch {
      return false;
    }
  }, [apiBaseUrl, makeHeaders, fetchStatus]);

  const disableTOTP = useCallback(async (): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/mfa/totp`, { method: 'DELETE', headers: makeHeaders() });
      if (!resp.ok) return false;
      await fetchStatus();
      return true;
    } catch {
      return false;
    }
  }, [apiBaseUrl, makeHeaders, fetchStatus]);

  const generateBackupCodes = useCallback(async (): Promise<BackupCodes | null> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/mfa/backup-codes`, { method: 'POST', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Generate failed (${resp.status})`);
      const data = await resp.json();
      await fetchStatus();
      return { codes: data.codes ?? [], generated_at: data.generated_at ?? new Date().toISOString() };
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return null;
    }
  }, [apiBaseUrl, makeHeaders, fetchStatus]);

  const registerWebAuthn = useCallback(async (name: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/webauthn/register`, {
        method: 'POST', headers: makeHeaders(), body: JSON.stringify({ name }),
      });
      if (!resp.ok) return false;
      await fetchStatus();
      return true;
    } catch {
      return false;
    }
  }, [apiBaseUrl, makeHeaders, fetchStatus]);

  const registerPlatformAuthenticator = useCallback(async (name: string): Promise<boolean> => {
    if (!isPlatformSupported) {
      setError('Platform authenticator not available on this device');
      return false;
    }
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/webauthn/register`, {
        method: 'POST', headers: makeHeaders(),
        body: JSON.stringify({ name, attachment: 'platform', user_verification: 'required' }),
      });
      if (!resp.ok) return false;
      await fetchStatus();
      return true;
    } catch {
      return false;
    }
  }, [apiBaseUrl, makeHeaders, fetchStatus, isPlatformSupported]);

  const unregisterWebAuthn = useCallback(async (id: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/webauthn/credentials/${id}`, {
        method: 'DELETE', headers: makeHeaders(),
      });
      if (!resp.ok) return false;
      await fetchStatus();
      return true;
    } catch {
      return false;
    }
  }, [apiBaseUrl, makeHeaders, fetchStatus]);

  // Check for platform authenticator support (WebAuthn conditional UI)
  useState(() => {
    if (typeof window !== 'undefined' && window.PublicKeyCredential) {
      window.PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable()
        .then(setIsPlatformSupported)
        .catch(() => setIsPlatformSupported(false));
    }
  });

  const platformCredentials = credentials.filter((c) => c.platform === 'platform' || c.device_type === 'platform');
  const crossPlatformCredentials = credentials.filter((c) => c.platform !== 'platform' && c.device_type !== 'platform');

  return {
    status, credentials, isLoading, error,
    enrollTOTP, verifyTOTP, disableTOTP,
    generateBackupCodes, registerWebAuthn, registerPlatformAuthenticator,
    unregisterWebAuthn,
    isPlatformSupported, platformCredentials, crossPlatformCredentials,
    fetchStatus,
  };
}
