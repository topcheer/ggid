/**
 * GGID React SDK — useEmailOTP hook
 *
 * Email OTP configuration, send, and verify.
 *
 * Usage:
 *   const { config, sendOTP, verifyOTP, updateConfig } = useEmailOTP();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface EmailOTPConfig {
  enabled: boolean;
  otp_length: number;
  expiry_seconds: number;
  rate_limit_per_hour: number;
  allowed_domains: string[];
  issuer_name: string;
}

export interface UseEmailOTPResult {
  config: EmailOTPConfig | null;
  isLoading: boolean;
  error: string | null;
  fetchConfig: () => Promise<void>;
  updateConfig: (patch: Partial<EmailOTPConfig>) => Promise<boolean>;
  sendOTP: (email: string) => Promise<boolean>;
  verifyOTP: (email: string, code: string) => Promise<boolean>;
}

export function useEmailOTP(): UseEmailOTPResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [config, setConfig] = useState<EmailOTPConfig | null>(null);
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
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/email-otp/config`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setConfig(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const updateConfig = useCallback(async (patch: Partial<EmailOTPConfig>): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/email-otp/config`, { method: 'PUT', headers: makeHeaders(), body: JSON.stringify(patch) });
      if (!resp.ok) throw new Error(`Update failed (${resp.status})`);
      const updated = await resp.json() as EmailOTPConfig;
      setConfig(updated);
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const sendOTP = useCallback(async (email: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/email-otp/send`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify({ email }) });
      if (!resp.ok) throw new Error(`Send failed (${resp.status})`);
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const verifyOTP = useCallback(async (email: string, code: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/email-otp/verify`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify({ email, code }) });
      if (!resp.ok) throw new Error(`Verify failed (${resp.status})`);
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { config, isLoading, error, fetchConfig, updateConfig, sendOTP, verifyOTP };
}
