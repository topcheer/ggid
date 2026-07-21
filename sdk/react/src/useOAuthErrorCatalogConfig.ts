import { useState, useCallback } from "react";

export interface ErrorCodeEntry {
  code: string;
  http_status: number;
  user_message: string;
  retry_guidance: string;
  developer_doc_url: string;
  severity: "info" | "warn" | "error" | "critical";
}

export interface CustomLocaleMessage {
  locale: string;
  messages: Record<string, string>;
}

export interface OAuthErrorCatalogConfig {
  error_codes: ErrorCodeEntry[];
  custom_error_messages_per_locale: CustomLocaleMessage[];
  troubleshooting_enabled: boolean;
}

export function useOAuthErrorCatalogConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<OAuthErrorCatalogConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-error-catalog-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<OAuthErrorCatalogConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-error-catalog-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
