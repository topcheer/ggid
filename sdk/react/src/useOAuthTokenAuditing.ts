import { useState, useCallback, useEffect } from "react";

export interface TokenAuditEntry {
  token_id: string;
  client: string;
  user: string;
  issued_at: string;
  scopes: string[];
  revoked_at: string | null;
  revoked_by: string | null;
  revoke_reason: string | null;
}

export interface SuspiciousPattern {
  pattern_type: string;
  severity: "low" | "medium" | "high" | "critical";
  description: string;
  count: number;
  last_seen: string;
}

export interface OAuthTokenAuditingData {
  audit_trail: TokenAuditEntry[];
  suspicious_patterns: SuspiciousPattern[];
}

export function useOAuthTokenAuditing() {
  const [data, setData] = useState<OAuthTokenAuditingData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        audit_trail: [
          { token_id: "tok-001", client: "client-web-001", user: "alice.chen", issued_at: "2026-01-15T10:00:00Z", scopes: ["openid", "profile", "read"], revoked_at: null, revoked_by: null, revoke_reason: null },
          { token_id: "tok-002", client: "client-mobile-002", user: "bob.martinez", issued_at: "2026-01-15T09:30:00Z", scopes: ["openid", "profile"], revoked_at: "2026-01-15T11:00:00Z", revoked_by: "admin.carol", revoke_reason: "Suspicious activity" },
          { token_id: "tok-003", client: "client-api-003", user: "service.bot", issued_at: "2026-01-15T08:00:00Z", scopes: ["read", "write"], revoked_at: null, revoked_by: null, revoke_reason: null },
          { token_id: "tok-004", client: "client-web-001", user: "dave.wilson", issued_at: "2026-01-14T16:00:00Z", scopes: ["openid", "profile", "admin"], revoked_at: "2026-01-14T18:00:00Z", revoked_by: "system.auto", revoke_reason: "Token reuse detected" },
          { token_id: "tok-005", client: "client-spa-005", user: "eve.brown", issued_at: "2026-01-14T14:00:00Z", scopes: ["openid", "profile"], revoked_at: null, revoked_by: null, revoke_reason: null },
        ],
        suspicious_patterns: [
          { pattern_type: "token_reuse", severity: "high", description: "Token used from 2 different IPs within 5 minutes", count: 3, last_seen: "1h ago" },
          { pattern_type: "scope_escalation", severity: "critical", description: "Token upgraded from read to admin scope without re-auth", count: 1, last_seen: "2h ago" },
          { pattern_type: "unusual_ip", severity: "medium", description: "Token issued from new geographic location", count: 5, last_seen: "30m ago" },
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
