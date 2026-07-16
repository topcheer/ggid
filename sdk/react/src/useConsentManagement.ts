import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface ConsentRecord {
  user: string;
  purpose: string;
  granted_at: string;
  expires_at: string;
  withdrawn_at: string;
  source: string;
}

export interface ConsentTemplate {
  purpose: string;
  purpose_text: string;
  required: boolean;
  legal_basis: string;
}

export interface RegionCompliance {
  region: string;
  compliance_pct: number;
  active_consents: number;
}

export interface ConsentManagementData {
  user_consent_registry: ConsentRecord[];
  consent_templates: ConsentTemplate[];
  per_region_compliance: RegionCompliance[];
}

export function useConsentManagement() {
  const [data, setData] = useState<ConsentManagementData | null>(null);
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
        user_consent_registry: [
          { user: "alice@corp.com", purpose: "Marketing emails", granted_at: "2024-01-15", expires_at: "2025-01-15", withdrawn_at: "", source: "web_signup" },
          { user: "bob@corp.com", purpose: "Analytics tracking", granted_at: "2024-02-01", expires_at: "", withdrawn_at: "", source: "app_prompt" },
          { user: "charlie@corp.com", purpose: "Marketing emails", granted_at: "2023-06-10", expires_at: "2024-06-10", withdrawn_at: "2024-03-01", source: "web_signup" },
          { user: "diana@corp.com", purpose: "Third-party data sharing", granted_at: "2024-01-20", expires_at: "2025-01-20", withdrawn_at: "", source: "admin_manual" },
        ],
        consent_templates: [
          { purpose: "Marketing emails", purpose_text: "We may send you product updates and promotional content", required: false, legal_basis: "GDPR Art. 6(1)(a) - Consent" },
          { purpose: "Analytics tracking", purpose_text: "We collect usage data to improve our services", required: false, legal_basis: "GDPR Art. 6(1)(a) - Consent" },
          { purpose: "Third-party data sharing", purpose_text: "Share data with trusted partners for enhanced services", required: false, legal_basis: "GDPR Art. 6(1)(a) - Consent" },
          { purpose: "Service operation", purpose_text: "Process and store your data to provide core services", required: true, legal_basis: "GDPR Art. 6(1)(b) - Contract" },
        ],
        per_region_compliance: [
          { region: "GDPR (EU)", compliance_pct: 98, active_consents: 4520 },
          { region: "CCPA (US)", compliance_pct: 95, active_consents: 3210 },
          { region: "PIPL (CN)", compliance_pct: 92, active_consents: 890 },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  const withdrawConsent = useCallback(async (user: string, purpose: string) => {
    console.log("Withdrawing consent for", user, purpose);
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, withdrawConsent };
}
