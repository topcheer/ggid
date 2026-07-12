import { useState, useCallback, useEffect } from "react";

export interface ScimMapping {
  id: string;
  external_group: string;
  local_role: string;
  auto_provision: boolean;
  sync_direction: string;
}

export interface ScimApp {
  app: string;
  mapping_count: number;
}

export interface LastSync {
  status: string;
  synced_at: string;
  added: number;
  removed: number;
  errors: number;
}

export interface ScimGroupMappingConfigData {
  mappings: ScimMapping[];
  per_app: ScimApp[];
  last_sync: LastSync;
}

export function useScimGroupMappingConfig() {
  const [data, setData] = useState<ScimGroupMappingConfigData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try { await new Promise((r) => setTimeout(r, 400));
      setData({ mappings: [
        { id: "s1", external_group: "Azure-Admin", local_role: "admin", auto_provision: true, sync_direction: "bidirectional" },
        { id: "s2", external_group: "GitHub-Devs", local_role: "developer", auto_provision: true, sync_direction: "inbound" },
        { id: "s3", external_group: "Okta-Readonly", local_role: "viewer", auto_provision: false, sync_direction: "inbound" },
      ], per_app: [
        { app: "Slack", mapping_count: 3 }, { app: "Jira", mapping_count: 2 }, { app: "GitHub", mapping_count: 5 },
      ], last_sync: { status: "success", synced_at: "5m ago", added: 2, removed: 0, errors: 0 } });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
