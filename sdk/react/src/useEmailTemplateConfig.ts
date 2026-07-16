import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface EmailTemplate {
  id: string;
  name: string;
  body_html: string;
  variables: string[];
  enabled: boolean;
}

export interface EmailTemplateConfigData {
  templates: EmailTemplate[];
}

export function useEmailTemplateConfig() {
  const [data, setData] = useState<EmailTemplateConfigData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try { // Try real API first
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
      setData({ templates: [
        { id: "welcome", name: "Welcome Email", body_html: "<h1>Welcome {{user_name}}!</h1><p>Your account is ready.</p>", variables: ["{{user_name}}"], enabled: true },
        { id: "password_reset", name: "Password Reset", body_html: "<h1>Reset Your Password</h1><p>Click <a href='{{reset_link}}'>here</a>.</p>", variables: ["{{user_name}}", "{{reset_link}}"], enabled: true },
        { id: "mfa_setup", name: "MFA Setup", body_html: "<h1>Set Up MFA</h1><p>Scan the QR code in your app.</p>", variables: ["{{user_name}}", "{{qr_code}}"], enabled: true },
        { id: "account_locked", name: "Account Locked", body_html: "<h1>Account Locked</h1><p>Too many failed attempts.</p>", variables: ["{{user_name}}", "{{unlock_link}}"], enabled: true },
        { id: "access_granted", name: "Access Granted", body_html: "<h1>New Access</h1><p>You now have {{role}} access.</p>", variables: ["{{user_name}}", "{{role}}"], enabled: false },
      ] });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
