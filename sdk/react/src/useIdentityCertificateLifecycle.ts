import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface Certificate {
  name: string;
  type: string;
  issuer: string;
  serial: string;
  valid_from: string;
  valid_to: string;
  days_to_expiry: number;
  auto_renew_days_before: number;
}

export interface RenewalItem {
  name: string;
  days_until_renewal: number;
  status: string;
}

export interface ExpiryItem {
  name: string;
  days_until: number;
}

export interface RevocationItem {
  serial: string;
  revoked_at: string;
  reason: string;
}

export interface IdentityCertificateLifecycleData {
  certificates: Certificate[];
  renewal_queue: RenewalItem[];
  expiry_calendar: ExpiryItem[];
  revocation_list: RevocationItem[];
}

export function useIdentityCertificateLifecycle() {
  const [data, setData] = useState<IdentityCertificateLifecycleData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
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
        certificates: [
          { name: "auth.ggid.dev TLS", type: "TLS", issuer: "Let's Encrypt", serial: "3B:8F:2A:1C:9D:4E:7B:0F:5A:2C:8E:1D:3F:4B:5C:6D", valid_from: "2025-06-01", valid_to: "2025-09-01", days_to_expiry: 45, auto_renew_days_before: 30 },
          { name: "JWT Signing Key", type: "JWT", issuer: "Internal CA", serial: "01:23:45:67:89:AB:CD:EF:FE:DC:BA:98:76:54:32:10", valid_from: "2025-01-01", valid_to: "2026-01-01", days_to_expiry: 180, auto_renew_days_before: 60 },
          { name: "mTLS Client Cert", type: "mTLS", issuer: "Internal CA", serial: "AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99", valid_from: "2025-05-15", valid_to: "2025-07-15", days_to_expiry: 12, auto_renew_days_before: 14 },
          { name: "SAML Signing", type: "signing", issuer: "DigiCert", serial: "FF:EE:DD:CC:BB:AA:99:88:77:66:55:44:33:22:11:00", valid_from: "2024-12-01", valid_to: "2026-12-01", days_to_expiry: 510, auto_renew_days_before: 90 },
        ],
        renewal_queue: [
          { name: "mTLS Client Cert", days_until_renewal: 0, status: "due_now" },
          { name: "auth.ggid.dev TLS", days_until_renewal: 15, status: "scheduled" },
        ],
        expiry_calendar: [
          { name: "mTLS Client Cert", days_until: 12 },
          { name: "auth.ggid.dev TLS", days_until: 45 },
          { name: "JWT Signing Key", days_until: 180 },
          { name: "SAML Signing", days_until: 510 },
        ],
        revocation_list: [
          { serial: "99:88:77:66:55:44:33:22:11:00:FF:EE:DD:CC:BB:AA", revoked_at: "3d ago", reason: "Key compromise" },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
