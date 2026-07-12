import { useState, useCallback } from "react";
export interface LdapConfig { server_url: string; bind_dn: string; base_dn: string; user_filter: string; group_filter: string; start_tls: boolean; attribute_mapping: { ldap_attr: string; local_attr: string }[]; sync_schedule: string; last_sync: string | null; auto_provision: boolean; }
export function useLdapSyncConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<LdapConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/ldap-config"); if (!res.ok) throw new Error("HTTP " + res.status); setConfig(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const saveConfig = useCallback(async (c: LdapConfig) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/ldap-config", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(c) }); if (!res.ok) throw new Error("HTTP " + res.status); setConfig(c); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  const testConnection = useCallback(async (c: LdapConfig) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/ldap-config/test", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(c) }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { config, loading, error, fetchConfig, saveConfig, testConnection };
}
