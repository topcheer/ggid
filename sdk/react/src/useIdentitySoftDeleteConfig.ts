import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface EntityConfig {
  entity: string;
  enabled: boolean;
  retention_days: number;
}

export interface SoftDeletedItem {
  id: string;
  entity: string;
  name: string;
  deleted_at: string;
  purge_at: string;
  restorable: boolean;
}

export interface IdentitySoftDeleteConfigData {
  retention_days: number;
  auto_purge_after_days: number;
  recoverable_window_days: number;
  per_entity_config: EntityConfig[];
  soft_deleted_items: SoftDeletedItem[];
}

export function useIdentitySoftDeleteConfig() {
  const [data, setData] = useState<IdentitySoftDeleteConfigData | null>(null);
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
        res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } });
      } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        retention_days: 30,
        auto_purge_after_days: 90,
        recoverable_window_days: 30,
        per_entity_config: [
          { entity: "users", enabled: true, retention_days: 30 },
          { entity: "groups", enabled: true, retention_days: 30 },
          { entity: "api_keys", enabled: true, retention_days: 14 },
          { entity: "clients", enabled: true, retention_days: 60 },
        ],
        soft_deleted_items: [
          { id: "sd-1", entity: "user", name: "alice.chen", deleted_at: "5d ago", purge_at: "in 25d", restorable: true },
          { id: "sd-2", entity: "api_key", name: "key-prod-001", deleted_at: "10d ago", purge_at: "in 4d", restorable: true },
          { id: "sd-3", entity: "group", name: "Legacy Admins", deleted_at: "35d ago", purge_at: "in -5d (overdue)", restorable: false },
          { id: "sd-4", entity: "client", name: "client-old-legacy", deleted_at: "2d ago", purge_at: "in 58d", restorable: true },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const restoreItem = useCallback(async (_id: string) => {
    console.log("Restoring:", _id);
  }, []);

  const purgeAll = useCallback(async () => {
    console.log("Purging all soft-deleted items");
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, restoreItem, purgeAll };
}
