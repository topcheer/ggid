import { useState, useCallback } from "react";

export interface ConnectionPoolConfig {
  min: number;
  max: number;
  idle_timeout_seconds: number;
}

export interface SearchOptimization {
  indexed_attributes: string[];
  query_cache_enabled: boolean;
  query_cache_ttl: number;
}

export interface GroupMembershipResolution {
  nested_depth: number;
  cache_ttl: number;
}

export interface DirectoryFederation {
  name: string;
  base_dn: string;
  bind_dn: string;
  priority: number;
}

export interface SyncTuning {
  batch_size: number;
  interval_seconds: number;
}

export interface LdapDirectoryConfig {
  connection_pool: ConnectionPoolConfig;
  search_optimization: SearchOptimization;
  group_membership_resolution: GroupMembershipResolution;
  multi_directory_federation: DirectoryFederation[];
  sync_tuning: SyncTuning;
}

export function useLdapDirectoryConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<LdapDirectoryConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/ldap-directory-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<LdapDirectoryConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/ldap-directory-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
