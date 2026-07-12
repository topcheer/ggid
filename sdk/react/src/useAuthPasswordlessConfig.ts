import { useState, useCallback, useEffect } from "react";

export interface PasswordlessMethod {
  method: "magic_link" | "passkey" | "webauthn" | "biometric";
  description: string;
  enabled: boolean;
}

export interface RoleRequirement {
  role: string;
  required_method: string;
  enforcement: "required" | "recommended" | "optional";
  grace_period_days: number;
}

export interface AuthPasswordlessConfigData {
  enabled_methods: PasswordlessMethod[];
  magic_link_expiry_minutes: number;
  passkey_rp_id: string;
  webauthn_timeout_seconds: number;
  fallback_to_password: boolean;
  per_role_requirement: RoleRequirement[];
}

export function useAuthPasswordlessConfig() {
  const [data, setData] = useState<AuthPasswordlessConfigData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        enabled_methods: [
          { method: "magic_link", description: "Send a one-time login link to user email", enabled: true },
          { method: "passkey", description: "FIDO2/WebAuthn passkeys (platform authenticators)", enabled: true },
          { method: "webauthn", description: "WebAuthn with security keys (YubiKey, Titan, etc.)", enabled: true },
          { method: "biometric", description: "Touch ID / Face ID via platform authenticator", enabled: false },
        ],
        magic_link_expiry_minutes: 15,
        passkey_rp_id: "idp.example.com",
        webauthn_timeout_seconds: 300,
        fallback_to_password: true,
        per_role_requirement: [
          { role: "admin", required_method: "webauthn", enforcement: "required", grace_period_days: 0 },
          { role: "security_analyst", required_method: "passkey", enforcement: "required", grace_period_days: 7 },
          { role: "developer", required_method: "passkey", enforcement: "recommended", grace_period_days: 30 },
          { role: "user", required_method: "magic_link", enforcement: "optional", grace_period_days: 90 },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData };
}
