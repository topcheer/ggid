import { useState, useCallback, useEffect } from "react";

export interface JwkKey {
  kid: string;
  alg: string;
  kty: string;
  created_at: string;
  status: "active" | "rotated" | "revoked";
}

export interface RotationRecord {
  old_kid: string;
  new_kid: string;
  timestamp: string;
  triggered_by: string;
  success: boolean;
}

export interface JwksUriHealth {
  uri: string;
  healthy: boolean;
  response_time_ms: number;
  cache_hit_rate: number;
}

export interface OAuthJwksManagementData {
  active_keys: JwkKey[];
  key_rotation_history: RotationRecord[];
  auto_rotation_interval_days: number;
  kid_strategy: string;
  jwks_uri_health: JwksUriHealth;
}

export function useOAuthJwksManagement() {
  const [data, setData] = useState<OAuthJwksManagementData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        active_keys: [
          { kid: "key-2026-01", alg: "RS256", kty: "RSA", created_at: "2026-01-10", status: "active" },
          { kid: "key-2025-12", alg: "RS256", kty: "RSA", created_at: "2025-12-01", status: "rotated" },
          { kid: "key-2025-11", alg: "RS256", kty: "RSA", created_at: "2025-11-01", status: "revoked" },
          { kid: "key-es-01", alg: "ES256", kty: "EC", created_at: "2026-01-10", status: "active" },
        ],
        key_rotation_history: [
          { old_kid: "key-2025-12", new_kid: "key-2026-01", timestamp: "2026-01-10 02:00 UTC", triggered_by: "auto-rotation", success: true },
          { old_kid: "key-2025-11", new_kid: "key-2025-12", timestamp: "2025-12-01 02:00 UTC", triggered_by: "auto-rotation", success: true },
          { old_kid: "key-2025-10", new_kid: "key-2025-11", timestamp: "2025-11-01 02:00 UTC", triggered_by: "manual", success: true },
        ],
        auto_rotation_interval_days: 30,
        kid_strategy: "x5t#S256",
        jwks_uri_health: {
          uri: "https://idp.example.com/.well-known/jwks.json",
          healthy: true,
          response_time_ms: 18,
          cache_hit_rate: 97,
        },
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const rotateKey = useCallback(async () => {
    console.log("Rotating key");
  }, []);

  const testEndpoint = useCallback(async () => {
    console.log("Testing JWKS endpoint");
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, rotateKey, testEndpoint };
}
