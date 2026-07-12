import { useState, useCallback, useEffect } from "react";

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
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
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
