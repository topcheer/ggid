import { useState, useCallback } from "react";

export interface TrustedCaCert {
  name: string;
  fingerprint: string;
  expiry: string;
}

export interface AuthMtlsConfig {
  require_mtls: boolean;
  trusted_ca_certs: TrustedCaCert[];
  per_client_cert_binding: boolean;
  revocation_check: "CRL" | "OCSP" | "both" | "none";
  allow_self_signed: boolean;
  fallback_to_bearer: boolean;
}

export function useAuthMtlsConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<AuthMtlsConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/auth-mtls-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<AuthMtlsConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/auth-mtls-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
