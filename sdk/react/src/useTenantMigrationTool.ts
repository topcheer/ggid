import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface MigrationScopeItem {
  name: string;
  record_count: number;
}

export interface DryRunResult {
  affected_records: number;
  estimated_duration: string;
  conflicts: number;
}

export interface MigrationRecord {
  id: string;
  timestamp: string;
  scope: string;
  status: string;
}

export interface TenantMigrationToolData {
  source_tenant: string;
  destination_tenant: string;
  migration_scope: MigrationScopeItem[];
  dry_run: DryRunResult;
  migration_history: MigrationRecord[];
}

export function useTenantMigrationTool() {
  const [data, setData] = useState<TenantMigrationToolData | null>(null);
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
        source_tenant: "tenant-staging (0000...0001)",
        destination_tenant: "tenant-prod (0000...0002)",
        migration_scope: [
          { name: "users", record_count: 4500 },
          { name: "groups", record_count: 120 },
          { name: "roles", record_count: 35 },
          { name: "policies", record_count: 80 },
          { name: "oauth_clients", record_count: 12 },
          { name: "audit_logs", record_count: 580000 },
        ],
        dry_run: { affected_records: 584747, estimated_duration: "~12 min", conflicts: 3 },
        migration_history: [
          { id: "m1", timestamp: "2026-06-15", scope: "users, roles", status: "completed" },
          { id: "m2", timestamp: "2026-05-01", scope: "all", status: "completed" },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  const executeMigration = useCallback((scope: string[]) => { console.log("Executing migration with scope", scope); }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, executeMigration };
}
