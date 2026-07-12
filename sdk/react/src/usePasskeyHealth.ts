import { useState, useCallback, useEffect } from "react";

export interface PasskeyInfo {
  id: string;
  user: string;
  device: string;
  platform: string;
  created_at: string;
  last_used: string;
  backup_eligible: boolean;
  backup_state: string;
}

export interface RecoveryOption {
  method: string;
  enabled: boolean;
}

export interface PasskeyHealthData {
  registered_passkeys: PasskeyInfo[];
  adoption_rate_pct: number;
  platform_distribution: { iOS: number; Android: number; Windows: number; macOS: number };
  stale_passkeys: PasskeyInfo[];
  recovery_options_config: RecoveryOption[];
}

export function usePasskeyHealth() {
  const [data, setData] = useState<PasskeyHealthData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        registered_passkeys: [
          { id: "pk-001", user: "alice@ggid.dev", device: "iPhone 15 Pro", platform: "iOS", created_at: "10d ago", last_used: "1m ago", backup_eligible: true, backup_state: "synced" },
          { id: "pk-002", user: "bob@ggid.dev", device: "Pixel 8", platform: "Android", created_at: "20d ago", last_used: "5m ago", backup_eligible: true, backup_state: "synced" },
          { id: "pk-003", user: "carol@ggid.dev", device: "MacBook Pro", platform: "macOS", created_at: "45d ago", last_used: "3h ago", backup_eligible: false, backup_state: "local" },
          { id: "pk-004", user: "dave@ggid.dev", device: "Surface Pro", platform: "Windows", created_at: "120d ago", last_used: "95d ago", backup_eligible: false, backup_state: "local" },
        ],
        adoption_rate_pct: 68,
        platform_distribution: { iOS: 45, Android: 28, Windows: 15, macOS: 12 },
        stale_passkeys: [
          { id: "pk-004", user: "dave@ggid.dev", device: "Surface Pro", platform: "Windows", created_at: "120d ago", last_used: "95d ago", backup_eligible: false, backup_state: "local" },
        ],
        recovery_options_config: [
          { method: "Account recovery key", enabled: true },
          { method: "Secondary passkey", enabled: true },
          { method: "Backup codes", enabled: false },
          { method: "Admin assistance", enabled: true },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
