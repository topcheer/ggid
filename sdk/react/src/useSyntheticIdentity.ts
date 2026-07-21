import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface FlaggedAccount {
  email: string;
  registration_source: string;
  disposable_domain: boolean;
  account_age_hours: number;
  risk_score: number;
}

export interface DetectionRule {
  rule_name: string;
  description: string;
  enabled: boolean;
}

export interface SyntheticIdentityData {
  flagged_accounts: FlaggedAccount[];
  disposable_domains_blocklist: string[];
  detection_rules: DetectionRule[];
  auto_block_enabled: boolean;
}

export function useSyntheticIdentity() {
  const [data, setData] = useState<SyntheticIdentityData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        flagged_accounts: [
          { email: "user1@tempmail.com", registration_source: "web_signup", disposable_domain: true, account_age_hours: 2, risk_score: 92 },
          { email: "fakeuser@guerrillamail.com", registration_source: "mobile_app", disposable_domain: true, account_age_hours: 5, risk_score: 88 },
          { email: "synthetic@10minutemail.com", registration_source: "api", disposable_domain: true, account_age_hours: 1, risk_score: 95 },
          { email: "real.maybe@protonmail.com", registration_source: "web_signup", disposable_domain: false, account_age_hours: 48, risk_score: 42 },
        ],
        disposable_domains_blocklist: ["tempmail.com", "guerrillamail.com", "10minutemail.com", "mailinator.com", "throwaway.email", "yopmail.com", "sharklasers.com", "dispostable.com"],
        detection_rules: [
          { rule_name: "Disposable domain check", description: "Block registrations from known disposable email providers", enabled: true },
          { rule_name: "Rapid registration detection", description: "Flag accounts created within 1 hour of IP first seen", enabled: true },
          { rule_name: "Phone verification gap", description: "Flag accounts without phone verification after 24h", enabled: false },
          { rule_name: "Synthetic SSN detection", description: "Check SSN against synthetic identity database", enabled: true },
        ],
        auto_block_enabled: false,
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
