import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface CertInfo {
  name: string;
  type: string;
  issuer: string;
  expiry_date: string;
  days_remaining: number;
  auto_renewal_enabled: boolean;
}

export interface AlertConfig {
  first_alert_days: number;
  escalation_days: number;
  channels: string[];
}

export interface CertExpiryTrackerData {
  certs: CertInfo[];
  alert_config: AlertConfig;
}

export function useCertExpiryTracker() {
  const [data, setData] = useState<CertExpiryTrackerData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", {
          headers: { "Content-Type": "application/json" },
        });
      } catch { res = null; }
      
      if (res?.ok) {
        const realData = await res.json();
        setData(realData);
        setIsDemoData(false);
        return;
      }
      
      // Fallback: empty demo data (no dangerous flags)
      setIsDemoData(true);
      setData({
        certs: [
          { name: "*.ggid.dev", type: "TLS", issuer: "Let's Encrypt", expiry_date: "2026-08-10", days_remaining: 29, auto_renewal_enabled: false },
          { name: "auth-signing-key", type: "JWT Signing", issuer: "Internal CA", expiry_date: "2026-09-15", days_remaining: 65, auto_renewal_enabled: false },
          { name: "oauth-mTLS", type: "mTLS", issuer: "Internal CA", expiry_date: "2026-07-20", days_remaining: 8, auto_renewal_enabled: false },
          { name: "saml-idp-cert", type: "SAML Signing", issuer: "DigiCert", expiry_date: "2026-12-01", days_remaining: 142, auto_renewal_enabled: false },
          { name: "gateway-tls", type: "TLS", issuer: "Let's Encrypt", expiry_date: "2026-07-14", days_remaining: 2, auto_renewal_enabled: false },
          { name: "ldap-tls", type: "TLS", issuer: "Internal CA", expiry_date: "2026-06-30", days_remaining: -14, auto_renewal_enabled: false },
        ],
        alert_config: { first_alert_days: 60, escalation_days: 14, channels: ["email", "slack", "pagerduty"] },
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
