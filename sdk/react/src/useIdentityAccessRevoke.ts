import { useState, useCallback, useEffect } from "react";

export interface SearchableUser {
  user_id: string;
  username: string;
  email: string;
  active: boolean;
}

export interface ImpactItem {
  category: string;
  count: number;
}

export interface LogEntry {
  action: string;
  timestamp: string;
  success: boolean;
}

export interface IdentityAccessRevokeData {
  searchable_users: SearchableUser[];
  estimated_impact: ImpactItem[];
  execution_log: LogEntry[];
}

export function useIdentityAccessRevoke() {
  const [data, setData] = useState<IdentityAccessRevokeData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        searchable_users: [
          { user_id: "u1", username: "alice.chen", email: "alice@ggid.dev", active: true },
          { user_id: "u2", username: "bob.martinez", email: "bob@ggid.dev", active: true },
          { user_id: "u3", username: "carol.jones", email: "carol@ggid.dev", active: true },
          { user_id: "u4", username: "dave.wilson", email: "dave@ggid.dev", active: false },
          { user_id: "u5", username: "eve.brown", email: "eve@ggid.dev", active: true },
        ],
        estimated_impact: [
          { category: "active_sessions", count: 4 },
          { category: "active_tokens", count: 12 },
          { category: "api_keys", count: 3 },
          { category: "app_access", count: 8 },
          { category: "ssh_keys", count: 2 },
        ],
        execution_log: [
          { action: "Revoked session sess-abc123", timestamp: "5m ago", success: true },
          { action: "Revoked token tok-def456", timestamp: "5m ago", success: true },
          { action: "Disabled API key key-ghi789", timestamp: "4m ago", success: true },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const executeRevoke = useCallback(async (_targets: string[], _reason: string, _notify: boolean) => {
    console.log("Revoking:", _targets, "Reason:", _reason, "Notify:", _notify);
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, executeRevoke };
}
