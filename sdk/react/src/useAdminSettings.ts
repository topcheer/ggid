import { useState, useCallback } from "react";
export interface AdminConfig { super_admins: { user_id: string; username: string; added_at: string; added_by: string }[]; permissions: Record<string, boolean>; restricted_actions: string[]; require_mfa: boolean; activity_log: { id: string; admin: string; action: string; target: string; timestamp: string }[]; }
export function useAdminSettings(baseUrl: string = "") {
  const [config, setConfig] = useState<AdminConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/admin/settings"); if (!res.ok) throw new Error("HTTP " + res.status); setConfig(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const saveConfig = useCallback(async (cfg: AdminConfig) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/admin/settings", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(cfg) }); if (!res.ok) throw new Error("HTTP " + res.status); setConfig(cfg); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { config, loading, error, fetchConfig, saveConfig };
}
