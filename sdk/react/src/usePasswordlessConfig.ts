import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface PasswordlessMethod {
  id: string;
  label: string;
  description: string;
  enabled: boolean;
}

export interface RoleRequirement {
  role: string;
  method: string;
}

export interface PasswordlessConfigData {
  methods: PasswordlessMethod[];
  magic_link_expiry_minutes: number;
  passkey_rp_id: string;
  fallback_to_password: boolean;
  per_role: RoleRequirement[];
}

export function usePasswordlessConfig() {
  const [data, setData] = useState<PasswordlessConfigData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        methods: [
          { id: "magic_link", label: "Magic Link", description: "Email-based one-click login", enabled: true },
          { id: "passkey", label: "Passkey", description: "Device-bound credential (FIDO2)", enabled: true },
          { id: "webauthn", label: "WebAuthn", description: "Hardware security key", enabled: true },
          { id: "biometric", label: "Biometric", description: "Face ID / Touch ID", enabled: false },
        ],
        magic_link_expiry_minutes: 15,
        passkey_rp_id: "auth.ggid.dev",
        fallback_to_password: true,
        per_role: [
          { role: "admin", method: "webauthn" },
          { role: "developer", method: "passkey" },
          { role: "user", method: "magic_link" },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
